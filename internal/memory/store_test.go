package memory

import (
	"errors"
	"testing"
	"time"
)

func TestStoreAppendGetList(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

	event, err := NewEvent("evt_1", "tenant_1", "session_1", 0, 10, now)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, ok := store.Get("tenant_1", "session_1", "evt_1")
	if !ok {
		t.Fatalf("expected event to exist")
	}
	if got.EventID != event.EventID {
		t.Fatalf("expected event id %q, got %q", event.EventID, got.EventID)
	}

	list := store.ListBySession("tenant_1", "session_1")
	if len(list) != 1 {
		t.Fatalf("expected one event, got %d", len(list))
	}
	if list[0].EventID != "evt_1" {
		t.Fatalf("expected event id %q, got %q", "evt_1", list[0].EventID)
	}
}

func TestStoreRejectsDuplicateEventIDInSession(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

	first, err := NewEvent("evt_1", "tenant_1", "session_1", 0, 10, now)
	if err != nil {
		t.Fatalf("new first event: %v", err)
	}
	second, err := NewEvent("evt_1", "tenant_1", "session_1", 10, 20, now.Add(time.Second))
	if err != nil {
		t.Fatalf("new second event: %v", err)
	}

	if err := store.Append(first); err != nil {
		t.Fatalf("append first: %v", err)
	}
	err = store.Append(second)
	if !errors.Is(err, ErrDuplicateEventID) {
		t.Fatalf("expected error %v, got %v", ErrDuplicateEventID, err)
	}
}

func TestStoreRejectsInvalidEvent(t *testing.T) {
	store := NewStore()
	err := store.Append(Event{})
	if !errors.Is(err, errEventIDRequired) {
		t.Fatalf("expected error %v, got %v", errEventIDRequired, err)
	}
}

func TestStoreAllowsSameEventIDAcrossSessionsAndTenants(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

	cases := []Event{
		mustEvent(t, "evt_1", "tenant_1", "session_1", 0, 10, now),
		mustEvent(t, "evt_1", "tenant_1", "session_2", 0, 10, now),
		mustEvent(t, "evt_1", "tenant_2", "session_1", 0, 10, now),
	}

	for _, event := range cases {
		if err := store.Append(event); err != nil {
			t.Fatalf("append event %v: %v", event, err)
		}
	}
}

func TestStoreListBySessionOrdering(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

	events := []Event{
		mustEvent(t, "evt_c", "tenant_1", "session_1", 20, 30, base.Add(2*time.Second)),
		mustEvent(t, "evt_b", "tenant_1", "session_1", 10, 20, base.Add(2*time.Second)),
		mustEvent(t, "evt_a", "tenant_1", "session_1", 10, 20, base.Add(1*time.Second)),
	}

	for _, event := range events {
		if err := store.Append(event); err != nil {
			t.Fatalf("append %q: %v", event.EventID, err)
		}
	}

	list := store.ListBySession("tenant_1", "session_1")
	if len(list) != 3 {
		t.Fatalf("expected three events, got %d", len(list))
	}

	expectedOrder := []string{"evt_a", "evt_b", "evt_c"}
	for i, eventID := range expectedOrder {
		if list[i].EventID != eventID {
			t.Fatalf("expected event %d to be %q, got %q", i, eventID, list[i].EventID)
		}
	}
}

func TestStoreIsolationAndNotFound(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)
	event := mustEvent(t, "evt_1", "tenant_1", "session_1", 0, 10, now)
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	if _, ok := store.Get("tenant_2", "session_1", "evt_1"); ok {
		t.Fatalf("expected no event for different tenant")
	}
	if _, ok := store.Get("tenant_1", "session_2", "evt_1"); ok {
		t.Fatalf("expected no event for different session")
	}
	if _, ok := store.Get("tenant_1", "session_1", "evt_2"); ok {
		t.Fatalf("expected no event for unknown id")
	}

	list := store.ListBySession("tenant_1", "session_2")
	if list == nil {
		t.Fatalf("expected empty list for unknown session, got nil")
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list for unknown session, got %#v", list)
	}
}

func TestStoreListBySessionReturnsCopy(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)
	event := mustEvent(t, "evt_1", "tenant_1", "session_1", 0, 10, now)
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	firstList := store.ListBySession("tenant_1", "session_1")
	firstList[0].EventID = "mutated"

	secondList := store.ListBySession("tenant_1", "session_1")
	if secondList[0].EventID != "evt_1" {
		t.Fatalf("expected store list to remain unchanged, got %q", secondList[0].EventID)
	}
}

func TestStoreAppendManyRejectsDuplicatesInBatch(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

	events := []Event{
		mustEvent(t, "evt_1", "tenant_1", "session_1", 0, 10, now),
		mustEvent(t, "evt_1", "tenant_1", "session_1", 10, 20, now.Add(time.Second)),
	}

	err := store.AppendMany(events)
	if !errors.Is(err, ErrDuplicateEventID) {
		t.Fatalf("expected error %v, got %v", ErrDuplicateEventID, err)
	}

	list := store.ListBySession("tenant_1", "session_1")
	if len(list) != 0 {
		t.Fatalf("expected no writes on batch failure, got %#v", list)
	}
}

func mustEvent(t *testing.T, eventID, tenantID, sessionID string, startToken, endTokenExclusive int, createdAt time.Time) Event {
	t.Helper()

	event, err := NewEvent(eventID, tenantID, sessionID, startToken, endTokenExclusive, createdAt)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	return event
}
