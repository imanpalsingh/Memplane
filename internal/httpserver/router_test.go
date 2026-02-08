package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router, err := NewRouter("test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if body := rec.Body.String(); body != "{\"status\":\"ok\"}" {
		t.Fatalf("expected body %q, got %q", "{\"status\":\"ok\"}", body)
	}
}
