package memory

import (
	"errors"
	"sort"
	"sync"
)

var ErrDuplicateEventID = errors.New("event_id already exists in tenant session")

var (
	errRetrieveTopKNonPositive = errors.New("top_k must be positive")
	errRetrieveBufferNegative  = errors.New("buffer_before and buffer_after must be non-negative")
)

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

func (s *Store) RetrieveByAnchors(
	tenantID, sessionID string,
	anchorEventIDs []string,
	topK, bufferBefore, bufferAfter int,
) ([]Event, error) {
	if topK <= 0 {
		return nil, errRetrieveTopKNonPositive
	}
	if bufferBefore < 0 || bufferAfter < 0 {
		return nil, errRetrieveBufferNegative
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey{tenantID: tenantID, sessionID: sessionID}
	session, ok := s.sessions[key]
	if !ok || len(session.ordered) == 0 {
		return []Event{}, nil
	}

	effectiveTopK := topK
	if effectiveTopK > len(anchorEventIDs) {
		effectiveTopK = len(anchorEventIDs)
	}
	if effectiveTopK > len(session.ordered) {
		effectiveTopK = len(session.ordered)
	}

	indexByID := make(map[string]int, len(session.ordered))
	for i, event := range session.ordered {
		indexByID[event.EventID] = i
	}

	anchorIndexes := make([]int, 0, effectiveTopK)
	seenAnchors := make(map[int]struct{}, effectiveTopK)
	for _, eventID := range anchorEventIDs {
		index, found := indexByID[eventID]
		if !found {
			continue
		}
		if _, exists := seenAnchors[index]; exists {
			continue
		}
		seenAnchors[index] = struct{}{}
		anchorIndexes = append(anchorIndexes, index)
		if len(anchorIndexes) == effectiveTopK {
			break
		}
	}

	if len(anchorIndexes) == 0 {
		return []Event{}, nil
	}

	includeIndexes := make(map[int]struct{})
	for _, anchor := range anchorIndexes {
		start := max(0, anchor-bufferBefore)
		end := min(len(session.ordered)-1, anchor+bufferAfter)
		for i := start; i <= end; i++ {
			includeIndexes[i] = struct{}{}
		}
	}

	orderedIndexes := make([]int, 0, len(includeIndexes))
	for i := range includeIndexes {
		orderedIndexes = append(orderedIndexes, i)
	}
	sort.Ints(orderedIndexes)

	result := make([]Event, 0, len(orderedIndexes))
	for _, i := range orderedIndexes {
		result = append(result, session.ordered[i])
	}

	return result, nil
}
