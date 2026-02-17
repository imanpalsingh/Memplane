package memory

import (
	"encoding/json"
	"errors"
	"math"
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

func TestNewEventRejectsInvalidRepresentativeKeys(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name               string
		representativeKeys [][]float64
		wantErr            error
	}{
		{
			name:               "empty key vector",
			representativeKeys: [][]float64{{}},
			wantErr:            errRepresentativeKeyEmpty,
		},
		{
			name:               "mismatched key dimensions",
			representativeKeys: [][]float64{{1, 2}, {3}},
			wantErr:            errRepresentativeKeyDimensionMismatch,
		},
		{
			name:               "zero norm key",
			representativeKeys: [][]float64{{0, 0}},
			wantErr:            errRepresentativeKeyZeroNorm,
		},
		{
			name:               "non finite key",
			representativeKeys: [][]float64{{1, math.Inf(1)}},
			wantErr:            errRepresentativeKeyNonFinite,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			event, err := NewEvent("evt_1", "tenant_1", "session_1", 1, 2, now)
			if err != nil {
				t.Fatalf("new event: %v", err)
			}

			event.RepresentativeKeys = tc.representativeKeys
			err = validateEvent(event)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestEventJSONShape(t *testing.T) {
	createdAt := time.Date(2026, 2, 8, 7, 0, 0, 0, time.UTC)
	event, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 20, createdAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if len(decoded) != 6 {
		t.Fatalf("expected 6 json fields, got %d", len(decoded))
	}
	if decoded["event_id"] != "evt_1" {
		t.Fatalf("expected event_id %q, got %#v", "evt_1", decoded["event_id"])
	}
	if decoded["tenant_id"] != "tenant_1" {
		t.Fatalf("expected tenant_id %q, got %#v", "tenant_1", decoded["tenant_id"])
	}
	if decoded["session_id"] != "session_1" {
		t.Fatalf("expected session_id %q, got %#v", "session_1", decoded["session_id"])
	}
	if decoded["start_token"] != float64(10) {
		t.Fatalf("expected start_token %v, got %#v", float64(10), decoded["start_token"])
	}
	if decoded["end_token_exclusive"] != float64(20) {
		t.Fatalf("expected end_token_exclusive %v, got %#v", float64(20), decoded["end_token_exclusive"])
	}

	createdAtRaw, ok := decoded["created_at"].(string)
	if !ok {
		t.Fatalf("expected created_at to be string, got %#v", decoded["created_at"])
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if !parsedCreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %v, got %v", createdAt, parsedCreatedAt)
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 2, 8, 7, 0, 0, 0, time.UTC)
	event, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 20, createdAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if decoded.EventID != event.EventID {
		t.Fatalf("expected event id %q, got %q", event.EventID, decoded.EventID)
	}
	if decoded.TenantID != event.TenantID {
		t.Fatalf("expected tenant id %q, got %q", event.TenantID, decoded.TenantID)
	}
	if decoded.SessionID != event.SessionID {
		t.Fatalf("expected session id %q, got %q", event.SessionID, decoded.SessionID)
	}
	if decoded.StartToken != event.StartToken {
		t.Fatalf("expected start token %d, got %d", event.StartToken, decoded.StartToken)
	}
	if decoded.EndTokenExclusive != event.EndTokenExclusive {
		t.Fatalf("expected end token exclusive %d, got %d", event.EndTokenExclusive, decoded.EndTokenExclusive)
	}
	if !decoded.CreatedAt.Equal(event.CreatedAt) {
		t.Fatalf("expected created_at %v, got %v", event.CreatedAt, decoded.CreatedAt)
	}
}

func TestEventJSONRoundTripWithRepresentativeKeys(t *testing.T) {
	createdAt := time.Date(2026, 2, 8, 7, 0, 0, 0, time.UTC)
	event, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 20, createdAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	event.RepresentativeKeys = [][]float64{{0.1, 0.9}, {0.2, 0.8}}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	if len(decoded.RepresentativeKeys) != 2 {
		t.Fatalf("expected 2 representative keys, got %d", len(decoded.RepresentativeKeys))
	}
	if len(decoded.RepresentativeKeys[0]) != 2 || len(decoded.RepresentativeKeys[1]) != 2 {
		t.Fatalf("unexpected representative key dimensions: %#v", decoded.RepresentativeKeys)
	}
}
