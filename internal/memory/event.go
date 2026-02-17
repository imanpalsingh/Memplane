package memory

import (
	"errors"
	"math"
	"time"
)

var (
	errEventIDRequired                    = errors.New("event_id is required")
	errTenantIDRequired                   = errors.New("tenant_id is required")
	errSessionIDRequired                  = errors.New("session_id is required")
	errStartTokenNegative                 = errors.New("start_token must be non-negative")
	errInvalidTokenRange                  = errors.New("end_token_exclusive must be greater than start_token")
	errCreatedAtRequired                  = errors.New("created_at is required")
	errRepresentativeKeyEmpty             = errors.New("representative_keys must not contain empty vectors")
	errRepresentativeKeyDimensionMismatch = errors.New("representative_keys vectors must all have the same dimension")
	errRepresentativeKeyZeroNorm          = errors.New("representative_keys must not contain zero-norm vectors")
	errRepresentativeKeyNonFinite         = errors.New("representative_keys must not contain NaN or Inf")
)

// Event represents one episodic memory segment in a tenant session.
// Token span uses a half-open interval: [StartToken, EndTokenExclusive).
type Event struct {
	EventID            string      `json:"event_id"`
	TenantID           string      `json:"tenant_id"`
	SessionID          string      `json:"session_id"`
	StartToken         int         `json:"start_token"`
	EndTokenExclusive  int         `json:"end_token_exclusive"`
	RepresentativeKeys [][]float64 `json:"representative_keys,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
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
	if err := validateRepresentativeKeys(event.RepresentativeKeys); err != nil {
		return err
	}

	return nil
}

func validateRepresentativeKeys(representativeKeys [][]float64) error {
	if len(representativeKeys) == 0 {
		return nil
	}

	expectedDimension := len(representativeKeys[0])
	if expectedDimension == 0 {
		return errRepresentativeKeyEmpty
	}

	for _, key := range representativeKeys {
		if len(key) == 0 {
			return errRepresentativeKeyEmpty
		}
		if len(key) != expectedDimension {
			return errRepresentativeKeyDimensionMismatch
		}

		normSquared := 0.0
		for _, value := range key {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return errRepresentativeKeyNonFinite
			}
			normSquared += value * value
		}
		if normSquared == 0 {
			return errRepresentativeKeyZeroNorm
		}
	}

	return nil
}
