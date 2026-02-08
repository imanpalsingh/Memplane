package memory

import (
	"errors"
	"sort"
	"sync"
)

var errDuplicateEventID = errors.New("event_id already exists in tenant session")

type Store struct {
	mu       sync.RWMutex
	sessions map[sessionKey]*sessionEvents
}

type sessionKey struct {
	tenantID  string
	sessionID string
}

type sessionEvents struct {
	ordered []Event
	byID    map[string]Event
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[sessionKey]*sessionEvents),
	}
}

func (s *Store) Append(event Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := sessionKey{tenantID: event.TenantID, sessionID: event.SessionID}
	events, ok := s.sessions[key]
	if !ok {
		events = &sessionEvents{
			ordered: make([]Event, 0, 1),
			byID:    make(map[string]Event),
		}
		s.sessions[key] = events
	}

	if _, exists := events.byID[event.EventID]; exists {
		return errDuplicateEventID
	}

	events.byID[event.EventID] = event
	events.ordered = append(events.ordered, event)

	sort.Slice(events.ordered, func(i, j int) bool {
		left := events.ordered[i]
		right := events.ordered[j]
		if left.StartToken != right.StartToken {
			return left.StartToken < right.StartToken
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.EventID < right.EventID
	})

	return nil
}

func (s *Store) Get(tenantID, sessionID, eventID string) (Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey{tenantID: tenantID, sessionID: sessionID}
	events, ok := s.sessions[key]
	if !ok {
		return Event{}, false
	}

	event, found := events.byID[eventID]
	if !found {
		return Event{}, false
	}

	return event, true
}

func (s *Store) ListBySession(tenantID, sessionID string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey{tenantID: tenantID, sessionID: sessionID}
	events, ok := s.sessions[key]
	if !ok {
		return []Event{}
	}

	list := make([]Event, len(events.ordered))
	copy(list, events.ordered)
	return list
}
