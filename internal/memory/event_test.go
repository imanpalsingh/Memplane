package memory

import (
	"errors"
	"testing"
	"time"
)

func TestNewEventValid(t *testing.T) {
	now := time.Now().UTC()

	event, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 20, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.EventID != "evt_1" {
		t.Fatalf("expected event id %q, got %q", "evt_1", event.EventID)
	}
	if event.TenantID != "tenant_1" {
		t.Fatalf("expected tenant id %q, got %q", "tenant_1", event.TenantID)
	}
	if event.SessionID != "session_1" {
		t.Fatalf("expected session id %q, got %q", "session_1", event.SessionID)
	}
	if event.StartToken != 10 {
		t.Fatalf("expected start token %d, got %d", 10, event.StartToken)
	}
	if event.EndTokenExclusive != 20 {
		t.Fatalf("expected end token exclusive %d, got %d", 20, event.EndTokenExclusive)
	}
	if !event.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, event.CreatedAt)
	}
}

func TestNewEventRejectsMissingEventID(t *testing.T) {
	_, err := NewEvent("", "tenant_1", "session_1", 1, 2, time.Now().UTC())
	if !errors.Is(err, errEventIDRequired) {
		t.Fatalf("expected error %v, got %v", errEventIDRequired, err)
	}
}

func TestNewEventRejectsMissingTenantID(t *testing.T) {
	_, err := NewEvent("evt_1", "", "session_1", 1, 2, time.Now().UTC())
	if !errors.Is(err, errTenantIDRequired) {
		t.Fatalf("expected error %v, got %v", errTenantIDRequired, err)
	}
}

func TestNewEventRejectsMissingSessionID(t *testing.T) {
	_, err := NewEvent("evt_1", "tenant_1", "", 1, 2, time.Now().UTC())
	if !errors.Is(err, errSessionIDRequired) {
		t.Fatalf("expected error %v, got %v", errSessionIDRequired, err)
	}
}

func TestNewEventRejectsNegativeStartToken(t *testing.T) {
	_, err := NewEvent("evt_1", "tenant_1", "session_1", -1, 2, time.Now().UTC())
	if !errors.Is(err, errStartTokenNegative) {
		t.Fatalf("expected error %v, got %v", errStartTokenNegative, err)
	}
}

func TestNewEventRejectsInvalidTokenRange(t *testing.T) {
	_, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 10, time.Now().UTC())
	if !errors.Is(err, errInvalidTokenRange) {
		t.Fatalf("expected error %v, got %v", errInvalidTokenRange, err)
	}
}

func TestNewEventRejectsMissingCreatedAt(t *testing.T) {
	_, err := NewEvent("evt_1", "tenant_1", "session_1", 1, 2, time.Time{})
	if !errors.Is(err, errCreatedAtRequired) {
		t.Fatalf("expected error %v, got %v", errCreatedAtRequired, err)
	}
}
