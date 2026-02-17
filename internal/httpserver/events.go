package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"memplane/internal/memory"

	"github.com/gin-gonic/gin"
)

type eventsHandler struct {
	store *memory.Store
}

type listEventsRequest struct {
	TenantID  string `form:"tenant_id" binding:"required"`
	SessionID string `form:"session_id" binding:"required"`
}

const maxJSONBodyBytes int64 = 1 << 20
const maxCreateEventBodyBytes int64 = maxJSONBodyBytes
const maxSegmentBodyBytes int64 = maxJSONBodyBytes
const maxRetrieveBodyBytes int64 = maxJSONBodyBytes
const maxSegmentSurpriseValues = 8192
const maxRetrieveAnchorEventIDs = 256
const maxRetrieveTopK = maxRetrieveAnchorEventIDs

var (
	errRequestBodyTooLarge = errors.New("request body too large")
	errInvalidRequestBody  = errors.New("invalid request body")
)

type segmentRequest struct {
	TenantID       string      `json:"tenant_id" binding:"required"`
	SessionID      string      `json:"session_id" binding:"required"`
	StartToken     int         `json:"start_token"`
	Surprise       []float64   `json:"surprise"`
	KeySimilarity  [][]float64 `json:"key_similarity" binding:"required"`
	Threshold      float64     `json:"threshold"`
	MinBoundaryGap int         `json:"min_boundary_gap"`
	CreatedAt      time.Time   `json:"created_at"`
	EventIDPrefix  string      `json:"event_id_prefix"`
}

type segmentResponse struct {
	Boundaries []int          `json:"boundaries"`
	Events     []memory.Event `json:"events"`
}

type retrieveRequest struct {
	TenantID     string   `json:"tenant_id" binding:"required"`
	SessionID    string   `json:"session_id" binding:"required"`
	EventIDs     []string `json:"event_ids"`
	TopK         int      `json:"top_k"`
	BufferBefore int      `json:"buffer_before"`
	BufferAfter  int      `json:"buffer_after"`
}

type retrieveResponse struct {
	Events []memory.Event `json:"events"`
}

func newEventsHandler(store *memory.Store) eventsHandler {
	return eventsHandler{store: store}
}

func (h eventsHandler) create(c *gin.Context) {
	var event memory.Event
	if err := bindJSONWithLimit(c, &event, maxCreateEventBodyBytes); err != nil {
		writeError(c, statusForBindError(err), err.Error())
		return
	}

	event.CreatedAt = event.CreatedAt.UTC()

	if err := h.store.Append(event); err != nil {
		if errors.Is(err, memory.ErrDuplicateEventID) {
			writeError(c, http.StatusConflict, err.Error())
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h eventsHandler) list(c *gin.Context) {
	var req listEventsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeError(c, http.StatusBadRequest, "tenant_id and session_id are required")
		return
	}

	events := h.store.ListBySession(req.TenantID, req.SessionID)
	c.JSON(http.StatusOK, events)
}

func (h eventsHandler) segment(c *gin.Context) {
	var req segmentRequest
	if err := bindJSONWithLimit(c, &req, maxSegmentBodyBytes); err != nil {
		writeError(c, statusForBindError(err), err.Error())
		return
	}

	req.CreatedAt = req.CreatedAt.UTC()
	if len(req.Surprise) > maxSegmentSurpriseValues {
		writeError(
			c,
			http.StatusBadRequest,
			fmt.Sprintf("surprise must contain at most %d values", maxSegmentSurpriseValues),
		)
		return
	}

	events, boundaries, err := memory.BuildEventsFromSurprise(
		req.TenantID,
		req.SessionID,
		req.StartToken,
		req.Surprise,
		req.KeySimilarity,
		req.Threshold,
		req.MinBoundaryGap,
		req.CreatedAt,
		req.EventIDPrefix,
	)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.AppendMany(events); err != nil {
		if errors.Is(err, memory.ErrDuplicateEventID) {
			writeError(c, http.StatusConflict, err.Error())
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, segmentResponse{
		Boundaries: boundaries,
		Events:     events,
	})
}

func (h eventsHandler) retrieve(c *gin.Context) {
	var req retrieveRequest
	if err := bindJSONWithLimit(c, &req, maxRetrieveBodyBytes); err != nil {
		writeError(c, statusForBindError(err), err.Error())
		return
	}

	if len(req.EventIDs) == 0 {
		writeError(c, http.StatusBadRequest, "event_ids must contain at least one event id")
		return
	}

	if len(req.EventIDs) > maxRetrieveAnchorEventIDs {
		writeError(
			c,
			http.StatusBadRequest,
			fmt.Sprintf("event_ids must contain at most %d items", maxRetrieveAnchorEventIDs),
		)
		return
	}
	if req.TopK > maxRetrieveTopK {
		writeError(c, http.StatusBadRequest, fmt.Sprintf("top_k must be at most %d", maxRetrieveTopK))
		return
	}
	for _, eventID := range req.EventIDs {
		if strings.TrimSpace(eventID) == "" {
			writeError(c, http.StatusBadRequest, "event_ids must not contain empty values")
			return
		}
	}

	events, err := h.store.RetrieveByAnchors(
		req.TenantID,
		req.SessionID,
		req.EventIDs,
		req.TopK,
		req.BufferBefore,
		req.BufferAfter,
	)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, retrieveResponse{Events: events})
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func bindJSONWithLimit(c *gin.Context, dst any, maxBodyBytes int64) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errRequestBodyTooLarge
		}
		return errInvalidRequestBody
	}
	return nil
}

func statusForBindError(err error) int {
	if errors.Is(err, errRequestBodyTooLarge) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}
