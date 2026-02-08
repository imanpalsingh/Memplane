package memory

import "time"

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
