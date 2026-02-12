package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	EnableStrictJSONDecoding()

	router, err := NewRouter("test", memory.NewStore())
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	return router
}
