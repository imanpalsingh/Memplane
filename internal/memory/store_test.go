package memory

import (
	"errors"
	"fmt"
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
	event := mustEventWithRepresentativeKeys(t, "evt_1", "tenant_1", "session_1", 0, 10, now, [][]float64{{0.1, 0.9}})
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	firstList := store.ListBySession("tenant_1", "session_1")
	firstList[0].EventID = "mutated"
	firstList[0].RepresentativeKeys[0][0] = 9.9

	secondList := store.ListBySession("tenant_1", "session_1")
	if secondList[0].EventID != "evt_1" {
		t.Fatalf("expected store list to remain unchanged, got %q", secondList[0].EventID)
	}
	if secondList[0].RepresentativeKeys[0][0] != 0.1 {
		t.Fatalf("expected representative keys to remain unchanged, got %#v", secondList[0].RepresentativeKeys)
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

func TestStoreRetrieveByAnchorsAppliesBuffers(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		event := mustEvent(
			t,
			fmt.Sprintf("evt_%d", i+1),
			"tenant_1",
			"session_1",
			i*10,
			(i+1)*10,
			base.Add(time.Duration(i)*time.Second),
		)
		if err := store.Append(event); err != nil {
			t.Fatalf("append %q: %v", event.EventID, err)
		}
	}

	events, err := store.RetrieveByAnchors("tenant_1", "session_1", []string{"evt_3"}, 1, 1, 1)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].EventID != "evt_2" || events[1].EventID != "evt_3" || events[2].EventID != "evt_4" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestStoreRetrieveByAnchorsHonorsTopKAndDeduplicates(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		event := mustEvent(
			t,
			fmt.Sprintf("evt_%d", i+1),
			"tenant_1",
			"session_1",
			i*10,
			(i+1)*10,
			base.Add(time.Duration(i)*time.Second),
		)
		if err := store.Append(event); err != nil {
			t.Fatalf("append %q: %v", event.EventID, err)
		}
	}

	events, err := store.RetrieveByAnchors("tenant_1", "session_1", []string{"evt_2", "evt_2", "evt_1"}, 1, 0, 0)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	if len(events) != 1 || events[0].EventID != "evt_2" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestStoreRetrieveByAnchorsValidation(t *testing.T) {
	store := NewStore()

	_, err := store.RetrieveByAnchors("tenant_1", "session_1", []string{"evt_1"}, 0, 0, 0)
	if !errors.Is(err, errRetrieveTopKNonPositive) {
		t.Fatalf("expected error %v, got %v", errRetrieveTopKNonPositive, err)
	}

	_, err = store.RetrieveByAnchors("tenant_1", "session_1", []string{"evt_1"}, 1, -1, 0)
	if !errors.Is(err, errRetrieveBufferNegative) {
		t.Fatalf("expected error %v, got %v", errRetrieveBufferNegative, err)
	}
}

func TestStoreRetrieveByQueryRanksAndAppliesBuffers(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC)
	events := []Event{
		mustEventWithRepresentativeKeys(t, "evt_1", "tenant_1", "session_1", 0, 10, base, [][]float64{{1, 0}}),
		mustEventWithRepresentativeKeys(t, "evt_2", "tenant_1", "session_1", 10, 20, base.Add(time.Second), [][]float64{{0.9, 0.1}}),
		mustEventWithRepresentativeKeys(t, "evt_3", "tenant_1", "session_1", 20, 30, base.Add(2*time.Second), [][]float64{{0, 1}}),
		mustEventWithRepresentativeKeys(t, "evt_4", "tenant_1", "session_1", 30, 40, base.Add(3*time.Second), [][]float64{{0.2, 0.98}}),
		mustEventWithRepresentativeKeys(t, "evt_5", "tenant_1", "session_1", 40, 50, base.Add(4*time.Second), [][]float64{{0.6, 0.8}}),
	}

	for _, event := range events {
		if err := store.Append(event); err != nil {
			t.Fatalf("append %q: %v", event.EventID, err)
		}
	}

	retrievedWithBuffer, err := store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 1, 1, 1)
	if err != nil {
		t.Fatalf("retrieve with buffer: %v", err)
	}
	if len(retrievedWithBuffer) != 3 {
		t.Fatalf("expected 3 events, got %d", len(retrievedWithBuffer))
	}
	if retrievedWithBuffer[0].EventID != "evt_2" || retrievedWithBuffer[1].EventID != "evt_3" || retrievedWithBuffer[2].EventID != "evt_4" {
		t.Fatalf("unexpected retrieval result: %#v", retrievedWithBuffer)
	}

	retrievedTop2, err := store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 2, 0, 0)
	if err != nil {
		t.Fatalf("retrieve top 2: %v", err)
	}
	if len(retrievedTop2) != 2 {
		t.Fatalf("expected 2 events, got %d", len(retrievedTop2))
	}
	if retrievedTop2[0].EventID != "evt_3" || retrievedTop2[1].EventID != "evt_4" {
		t.Fatalf("unexpected top 2 retrieval result: %#v", retrievedTop2)
	}
}

func TestStoreRetrieveByQuerySkipsEventsWithoutRepresentativeKeys(t *testing.T) {
	store := NewStore()
	event := mustEvent(t, "evt_1", "tenant_1", "session_1", 0, 10, time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC))
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 1, 0, 0)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no results, got %#v", events)
	}
}

func TestStoreRetrieveByQueryValidation(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC)

	validEvent := mustEventWithRepresentativeKeys(
		t,
		"evt_1",
		"tenant_1",
		"session_1",
		0,
		10,
		base,
		[][]float64{{1, 0}},
	)
	if err := store.Append(validEvent); err != nil {
		t.Fatalf("append valid event: %v", err)
	}

	_, err := store.RetrieveByQuery("tenant_1", "session_1", nil, 1, 0, 0)
	if !errors.Is(err, errRetrieveQueryVectorRequired) {
		t.Fatalf("expected error %v, got %v", errRetrieveQueryVectorRequired, err)
	}

	_, err = store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 0}, 1, 0, 0)
	if !errors.Is(err, errRetrieveQueryVectorZeroNorm) {
		t.Fatalf("expected error %v, got %v", errRetrieveQueryVectorZeroNorm, err)
	}

	_, err = store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 0, 0, 0)
	if !errors.Is(err, errRetrieveTopKNonPositive) {
		t.Fatalf("expected error %v, got %v", errRetrieveTopKNonPositive, err)
	}

	_, err = store.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 1, -1, 0)
	if !errors.Is(err, errRetrieveBufferNegative) {
		t.Fatalf("expected error %v, got %v", errRetrieveBufferNegative, err)
	}

	mismatchStore := NewStore()
	mismatchedEvent := mustEventWithRepresentativeKeys(
		t,
		"evt_mismatch",
		"tenant_1",
		"session_1",
		0,
		10,
		base,
		[][]float64{{1, 0, 0}},
	)
	if err := mismatchStore.Append(mismatchedEvent); err != nil {
		t.Fatalf("append mismatched event: %v", err)
	}
	_, err = mismatchStore.RetrieveByQuery("tenant_1", "session_1", []float64{0, 1}, 1, 0, 0)
	if !errors.Is(err, errRetrieveQueryVectorDimensionMismatch) {
		t.Fatalf("expected error %v, got %v", errRetrieveQueryVectorDimensionMismatch, err)
	}
}

func TestStoreAppendClonesRepresentativeKeys(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC)

	event := mustEventWithRepresentativeKeys(
		t,
		"evt_1",
		"tenant_1",
		"session_1",
		0,
		10,
		base,
		[][]float64{{0.1, 0.9}},
	)
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	event.RepresentativeKeys[0][0] = 7.7

	got, ok := store.Get("tenant_1", "session_1", "evt_1")
	if !ok {
		t.Fatalf("expected event to exist")
	}
	if got.RepresentativeKeys[0][0] != 0.1 {
		t.Fatalf("expected stored representative keys to be unchanged, got %#v", got.RepresentativeKeys)
	}
}

func TestStoreGetReturnsDeepCopy(t *testing.T) {
	store := NewStore()
	base := time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC)

	event := mustEventWithRepresentativeKeys(
		t,
		"evt_1",
		"tenant_1",
		"session_1",
		0,
		10,
		base,
		[][]float64{{0.1, 0.9}},
	)
	if err := store.Append(event); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, ok := store.Get("tenant_1", "session_1", "evt_1")
	if !ok {
		t.Fatalf("expected event to exist")
	}
	got.RepresentativeKeys[0][0] = 8.8

	gotAgain, ok := store.Get("tenant_1", "session_1", "evt_1")
	if !ok {
		t.Fatalf("expected event to exist")
	}
	if gotAgain.RepresentativeKeys[0][0] != 0.1 {
		t.Fatalf("expected representative keys to be unchanged, got %#v", gotAgain.RepresentativeKeys)
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

func mustEventWithRepresentativeKeys(
	t *testing.T,
	eventID, tenantID, sessionID string,
	startToken, endTokenExclusive int,
	createdAt time.Time,
	representativeKeys [][]float64,
) Event {
	t.Helper()

	event := mustEvent(t, eventID, tenantID, sessionID, startToken, endTokenExclusive, createdAt)
	event.RepresentativeKeys = representativeKeys
	return event
}
