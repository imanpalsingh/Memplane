package memory

import (
	"errors"
	"sort"
	"sync"
)

var ErrDuplicateEventID = errors.New("event_id already exists in tenant session")

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
	return s.AppendMany([]Event{event})
}

func (s *Store) AppendMany(events []Event) error {
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		if err := validateEvent(event); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seenBySession := make(map[sessionKey]map[string]struct{})
	for _, event := range events {
		key := sessionKey{tenantID: event.TenantID, sessionID: event.SessionID}
		if existingSession, ok := s.sessions[key]; ok {
			if _, exists := existingSession.byID[event.EventID]; exists {
				return ErrDuplicateEventID
			}
		}

		seenIDs, ok := seenBySession[key]
		if !ok {
			seenIDs = make(map[string]struct{})
			seenBySession[key] = seenIDs
		}
		if _, exists := seenIDs[event.EventID]; exists {
			return ErrDuplicateEventID
		}
		seenIDs[event.EventID] = struct{}{}
	}

	updatedSessions := make(map[sessionKey]struct{})
	for _, event := range events {
		key := sessionKey{tenantID: event.TenantID, sessionID: event.SessionID}
		session := s.ensureSession(key)
		session.byID[event.EventID] = event
		session.ordered = append(session.ordered, event)
		updatedSessions[key] = struct{}{}
	}

	for key := range updatedSessions {
		sortSessionEvents(s.sessions[key])
	}

	return nil
}

func (s *Store) ensureSession(key sessionKey) *sessionEvents {
	events, ok := s.sessions[key]
	if ok {
		return events
	}

	events = &sessionEvents{
		ordered: make([]Event, 0, 1),
		byID:    make(map[string]Event),
	}
	s.sessions[key] = events
	return events
}

func sortSessionEvents(events *sessionEvents) {
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
