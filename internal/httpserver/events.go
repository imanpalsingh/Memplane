package httpserver

import (
	"errors"
	"net/http"

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

const maxCreateEventBodyBytes int64 = 1 << 20

func newEventsHandler(store *memory.Store) eventsHandler {
	return eventsHandler{store: store}
}

func (h eventsHandler) create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCreateEventBodyBytes)

	var event memory.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(c, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(c, http.StatusBadRequest, "invalid request body")
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

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
