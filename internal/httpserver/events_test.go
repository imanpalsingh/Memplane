package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"memplane/internal/memory"
)

func TestCreateEventSuccess(t *testing.T) {
	router := newTestRouter(t)

	body := `{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var event memory.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &event); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if event.EventID != "evt_1" {
		t.Fatalf("expected event id %q, got %q", "evt_1", event.EventID)
	}
}

func TestCreateEventRejectsDuplicate(t *testing.T) {
	router := newTestRouter(t)

	body := `{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("expected first status %d, got %d", http.StatusCreated, rec.Code)
		}
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("expected second status %d, got %d", http.StatusConflict, rec.Code)
		}
	}
}

func TestCreateEventRejectsUnknownField(t *testing.T) {
	router := newTestRouter(t)

	body := `{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z","unexpected":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateEventRejectsInvalidTokenRange(t *testing.T) {
	router := newTestRouter(t)

	body := `{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":10,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateEventRejectsMissingCreatedAt(t *testing.T) {
	router := newTestRouter(t)

	body := `{"event_id":"evt_1","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateEventRejectsOversizedBody(t *testing.T) {
	router := newTestRouter(t)

	tooLargeEventID := strings.Repeat("a", int(maxCreateEventBodyBytes))
	body := fmt.Sprintf(
		`{"event_id":"%s","tenant_id":"tenant_1","session_id":"session_1","start_token":0,"end_token_exclusive":10,"created_at":"2026-02-10T12:00:00Z"}`,
		tooLargeEventID,
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestListEventsSuccess(t *testing.T) {
	store := memory.NewStore()
	router, err := NewRouter("test", store)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	base := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	first, err := memory.NewEvent("evt_2", "tenant_1", "session_1", 10, 20, base.Add(time.Second))
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	second, err := memory.NewEvent("evt_1", "tenant_1", "session_1", 0, 10, base)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	if err := store.Append(first); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := store.Append(second); err != nil {
		t.Fatalf("append event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/events?tenant_id=tenant_1&session_id=session_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var events []memory.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventID != "evt_1" || events[1].EventID != "evt_2" {
		t.Fatalf("unexpected order: %#v", events)
	}
}

func TestListEventsReturnsEmptyArray(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/events?tenant_id=tenant_1&session_id=session_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body != "[]" {
		t.Fatalf("expected body %q, got %q", "[]", body)
	}
}

func TestListEventsRejectsMissingQuery(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSegmentSuccess(t *testing.T) {
	store := memory.NewStore()
	router, err := NewRouter("test", store)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	body := `{
		"tenant_id":"tenant_1",
		"session_id":"session_1",
		"start_token":100,
		"surprise":[0.05,0.2,1.2,0.1,0.15,1.5,0.2],
		"key_similarity":[
			[1,0,0,0,0,0,0],
			[0,1,0,0,0,0,0],
			[0,0,1,0,0,0,0],
			[0,0,0,1,0,0,0],
			[0,0,0,0,1,0,0],
			[0,0,0,0,0,1,0],
			[0,0,0,0,0,0,1]
		],
		"threshold":0.8,
		"min_boundary_gap":1,
		"created_at":"2026-02-14T12:00:00Z",
		"event_id_prefix":"seg"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp struct {
		Boundaries []int          `json:"boundaries"`
		Events     []memory.Event `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(resp.Boundaries) != 2 || resp.Boundaries[0] != 103 || resp.Boundaries[1] != 105 {
		t.Fatalf("unexpected boundaries: %#v", resp.Boundaries)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}

	stored := store.ListBySession("tenant_1", "session_1")
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored events, got %d", len(stored))
	}
}

func TestSegmentRejectsTooManySurpriseValues(t *testing.T) {
	router := newTestRouter(t)

	surprise := make([]float64, maxSegmentSurpriseValues+1)
	for i := range surprise {
		surprise[i] = 0.1
	}

	body, err := json.Marshal(segmentRequest{
		TenantID:       "tenant_1",
		SessionID:      "session_1",
		StartToken:     100,
		Surprise:       surprise,
		KeySimilarity:  [][]float64{{1}},
		Threshold:      0.8,
		MinBoundaryGap: 1,
		CreatedAt:      time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC),
		EventIDPrefix:  "seg",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSegmentUsesBoundaryRefinementWhenSimilarityProvided(t *testing.T) {
	router := newTestRouter(t)

	body := `{
		"tenant_id":"tenant_1",
		"session_id":"session_1",
		"start_token":0,
		"surprise":[0.05,0.2,1.2,0.1,0.05,0.02],
		"key_similarity":[
			[1,4,0.1,0.1,0.1,0.1],
			[4,1,0.1,0.1,0.1,0.1],
			[0.1,0.1,1,4,4,4],
			[0.1,0.1,4,1,4,4],
			[0.1,0.1,4,4,1,4],
			[0.1,0.1,4,4,4,1]
		],
		"threshold":0.8,
		"min_boundary_gap":1,
		"created_at":"2026-02-14T12:00:00Z",
		"event_id_prefix":"seg_ref"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp struct {
		Boundaries []int `json:"boundaries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Boundaries) != 1 || resp.Boundaries[0] != 2 {
		t.Fatalf("expected refined boundary [2], got %#v", resp.Boundaries)
	}
}

func TestSegmentRejectsMissingKeySimilarity(t *testing.T) {
	router := newTestRouter(t)

	body := `{
		"tenant_id":"tenant_1",
		"session_id":"session_1",
		"start_token":100,
		"surprise":[0.05,0.2,1.2,0.1,0.15,1.5,0.2],
		"threshold":0.8,
		"min_boundary_gap":1,
		"created_at":"2026-02-14T12:00:00Z",
		"event_id_prefix":"seg"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSegmentRejectsNonSquareKeySimilarity(t *testing.T) {
	router := newTestRouter(t)

	body := `{
		"tenant_id":"tenant_1",
		"session_id":"session_1",
		"start_token":100,
		"surprise":[0.05,0.2],
		"key_similarity":[
			[1,0.1],
			[0.1]
		],
		"threshold":0.8,
		"min_boundary_gap":1,
		"created_at":"2026-02-14T12:00:00Z",
		"event_id_prefix":"seg"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSegmentRejectsDuplicatePrefixWithoutPartialWrites(t *testing.T) {
	store := memory.NewStore()
	router, err := NewRouter("test", store)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	body := `{
		"tenant_id":"tenant_1",
		"session_id":"session_1",
		"start_token":100,
		"surprise":[0.05,0.2,1.2,0.1,0.15,1.5,0.2],
		"key_similarity":[
			[1,0,0,0,0,0,0],
			[0,1,0,0,0,0,0],
			[0,0,1,0,0,0,0],
			[0,0,0,1,0,0,0],
			[0,0,0,0,1,0,0],
			[0,0,0,0,0,1,0],
			[0,0,0,0,0,0,1]
		],
		"threshold":0.8,
		"min_boundary_gap":1,
		"created_at":"2026-02-14T12:00:00Z",
		"event_id_prefix":"seg"
	}`

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/segment", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("expected first status %d, got %d", http.StatusCreated, rec.Code)
		}
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("expected second status %d, got %d", http.StatusConflict, rec.Code)
		}
	}

	stored := store.ListBySession("tenant_1", "session_1")
	if len(stored) != 3 {
		t.Fatalf("expected store to remain at 3 events, got %d", len(stored))
	}
}

func TestRetrieveSuccessWithBuffers(t *testing.T) {
	store := memory.NewStore()
	base := time.Date(2026, 2, 15, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		event, err := memory.NewEvent(
			fmt.Sprintf("evt_%d", i+1),
			"tenant_1",
			"session_1",
			i*10,
			(i+1)*10,
			base.Add(time.Duration(i)*time.Second),
		)
		if err != nil {
			t.Fatalf("new event: %v", err)
		}
		if err := store.Append(event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	router, err := NewRouter("test", store)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	body := `{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["evt_3"],"top_k":1,"buffer_before":1,"buffer_after":1}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp struct {
		Events []memory.Event `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}
	if resp.Events[0].EventID != "evt_2" || resp.Events[1].EventID != "evt_3" || resp.Events[2].EventID != "evt_4" {
		t.Fatalf("unexpected events: %#v", resp.Events)
	}
}

func TestRetrieveRejectsInvalidRequest(t *testing.T) {
	router := newTestRouter(t)

	body := `{"tenant_id":"tenant_1","session_id":"session_1","event_ids":[],"top_k":1,"buffer_before":0,"buffer_after":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRetrieveRejectsInvalidTopK(t *testing.T) {
	router := newTestRouter(t)

	body := `{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["evt_1"],"top_k":0,"buffer_before":0,"buffer_after":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRetrieveRejectsTooLargeTopK(t *testing.T) {
	router := newTestRouter(t)

	body := fmt.Sprintf(
		`{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["evt_1"],"top_k":%d,"buffer_before":0,"buffer_after":0}`,
		maxRetrieveTopK+1,
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRetrieveRejectsEmptyEventIDValues(t *testing.T) {
	router := newTestRouter(t)

	body := `{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["  "],"top_k":1,"buffer_before":0,"buffer_after":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRetrieveRejectsNegativeBuffers(t *testing.T) {
	router := newTestRouter(t)

	body := `{"tenant_id":"tenant_1","session_id":"session_1","event_ids":["evt_1"],"top_k":1,"buffer_before":-1,"buffer_after":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRetrieveRejectsTooManyAnchorEventIDs(t *testing.T) {
	router := newTestRouter(t)

	eventIDs := make([]string, maxRetrieveAnchorEventIDs+1)
	for i := range eventIDs {
		eventIDs[i] = fmt.Sprintf("evt_%d", i+1)
	}
	bodyBytes, err := json.Marshal(retrieveRequest{
		TenantID:     "tenant_1",
		SessionID:    "session_1",
		EventIDs:     eventIDs,
		TopK:         1,
		BufferBefore: 0,
		BufferAfter:  0,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/retrieve", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	router, err := NewRouter("test", memory.NewStore())
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	return router
}
