package memory

import (
	"errors"
	"time"
)

var (
	errEventIDRequired    = errors.New("event_id is required")
	errTenantIDRequired   = errors.New("tenant_id is required")
	errSessionIDRequired  = errors.New("session_id is required")
	errStartTokenNegative = errors.New("start_token must be non-negative")
	errInvalidTokenRange  = errors.New("end_token_exclusive must be greater than start_token")
	errCreatedAtRequired  = errors.New("created_at is required")
)

// Event represents one episodic memory segment in a tenant session.
// Token span uses a half-open interval: [StartToken, EndTokenExclusive).
type Event struct {
	EventID           string    `json:"event_id"`
	TenantID          string    `json:"tenant_id"`
	SessionID         string    `json:"session_id"`
	StartToken        int       `json:"start_token"`
	EndTokenExclusive int       `json:"end_token_exclusive"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewEvent(eventID, tenantID, sessionID string, startToken, endTokenExclusive int, createdAt time.Time) (Event, error) {
	event := Event{
		EventID:           eventID,
		TenantID:          tenantID,
		SessionID:         sessionID,
		StartToken:        startToken,
		EndTokenExclusive: endTokenExclusive,
		CreatedAt:         createdAt,
	}

	if err := validateEvent(event); err != nil {
		return Event{}, err
	}

	return event, nil
}

func validateEvent(event Event) error {
	if event.EventID == "" {
		return errEventIDRequired
	}
	if event.TenantID == "" {
		return errTenantIDRequired
	}
	if event.SessionID == "" {
		return errSessionIDRequired
	}
	if event.StartToken < 0 {
		return errStartTokenNegative
	}
	if event.EndTokenExclusive <= event.StartToken {
		return errInvalidTokenRange
	}
	if event.CreatedAt.IsZero() {
		return errCreatedAtRequired
	}

	return nil
}
