package memory

import (
	"errors"
	"math"
	"sort"
	"sync"
)

var ErrDuplicateEventID = errors.New("event_id already exists in tenant session")

var (
	errRetrieveTopKNonPositive              = errors.New("top_k must be positive")
	errRetrieveBufferNegative               = errors.New("buffer_before and buffer_after must be non-negative")
	errRetrieveQueryVectorRequired          = errors.New("query_vector must contain at least one value")
	errRetrieveQueryVectorNonFinite         = errors.New("query_vector must not contain NaN or Inf")
	errRetrieveQueryVectorZeroNorm          = errors.New("query_vector must not be a zero-norm vector")
	errRetrieveQueryVectorDimensionMismatch = errors.New("query_vector dimension must match representative_keys dimension")
	errRetrieveRepresentativeKeyZeroNorm    = errors.New("representative_keys must not contain zero-norm vectors")
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

	validatedEvents := make([]Event, len(events))
	for i, event := range events {
		if err := validateEvent(event); err != nil {
			return err
		}
		validatedEvents[i] = cloneEvent(event)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seenBySession := make(map[sessionKey]map[string]struct{})
	for _, event := range validatedEvents {
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
	for _, event := range validatedEvents {
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

	return cloneEvent(event), true
}

func (s *Store) ListBySession(tenantID, sessionID string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey{tenantID: tenantID, sessionID: sessionID}
	events, ok := s.sessions[key]
	if !ok {
		return []Event{}
	}

	list := make([]Event, 0, len(events.ordered))
	for _, event := range events.ordered {
		list = append(list, cloneEvent(event))
	}
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

	effectiveTopK := min(topK, len(anchorEventIDs), len(session.ordered))
	anchorIndexes := anchorIndexesFromEventIDs(session.ordered, anchorEventIDs, effectiveTopK)
	if len(anchorIndexes) == 0 {
		return []Event{}, nil
	}

	return bufferedEventsByAnchorIndexes(session.ordered, anchorIndexes, bufferBefore, bufferAfter), nil
}

func (s *Store) RetrieveByQuery(
	tenantID, sessionID string,
	queryVector []float64,
	topK, bufferBefore, bufferAfter int,
) ([]Event, error) {
	if topK <= 0 {
		return nil, errRetrieveTopKNonPositive
	}
	if bufferBefore < 0 || bufferAfter < 0 {
		return nil, errRetrieveBufferNegative
	}

	queryNorm, err := validateQueryVector(queryVector)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := sessionKey{tenantID: tenantID, sessionID: sessionID}
	session, ok := s.sessions[key]
	if !ok || len(session.ordered) == 0 {
		return []Event{}, nil
	}

	effectiveTopK := min(topK, len(session.ordered))
	anchorIndexes, err := anchorIndexesByQuery(session.ordered, queryVector, queryNorm, effectiveTopK)
	if err != nil {
		return nil, err
	}
	if len(anchorIndexes) == 0 {
		return []Event{}, nil
	}

	return bufferedEventsByAnchorIndexes(session.ordered, anchorIndexes, bufferBefore, bufferAfter), nil
}

type scoredAnchor struct {
	index int
	score float64
}

func anchorIndexesFromEventIDs(ordered []Event, anchorEventIDs []string, topK int) []int {
	indexByID := make(map[string]int, len(ordered))
	for i, event := range ordered {
		indexByID[event.EventID] = i
	}

	anchorIndexes := make([]int, 0, topK)
	seenAnchors := make(map[int]struct{}, topK)
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
		if len(anchorIndexes) == topK {
			break
		}
	}

	return anchorIndexes
}

func anchorIndexesByQuery(ordered []Event, queryVector []float64, queryNorm float64, topK int) ([]int, error) {
	candidates := make([]scoredAnchor, 0, len(ordered))
	for i, event := range ordered {
		score, found, err := eventQueryScore(event, queryVector, queryNorm)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		candidates = append(candidates, scoredAnchor{index: i, score: score})
	}

	if len(candidates) == 0 {
		return []int{}, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.score == right.score {
			return left.index < right.index
		}
		return left.score > right.score
	})

	effectiveTopK := min(topK, len(candidates))
	anchorIndexes := make([]int, 0, effectiveTopK)
	for i := 0; i < effectiveTopK; i++ {
		anchorIndexes = append(anchorIndexes, candidates[i].index)
	}
	return anchorIndexes, nil
}

func eventQueryScore(event Event, queryVector []float64, queryNorm float64) (float64, bool, error) {
	if len(event.RepresentativeKeys) == 0 {
		return 0, false, nil
	}

	bestScore := 0.0
	found := false
	for _, representativeKey := range event.RepresentativeKeys {
		if len(representativeKey) != len(queryVector) {
			return 0, false, errRetrieveQueryVectorDimensionMismatch
		}

		keyNorm := vectorNorm(representativeKey)
		if keyNorm == 0 {
			return 0, false, errRetrieveRepresentativeKeyZeroNorm
		}

		score := dotProduct(queryVector, representativeKey) / (queryNorm * keyNorm)
		if !found || score > bestScore {
			bestScore = score
			found = true
		}
	}

	return bestScore, true, nil
}

func bufferedEventsByAnchorIndexes(ordered []Event, anchorIndexes []int, bufferBefore, bufferAfter int) []Event {
	includeIndexes := make(map[int]struct{})
	for _, anchor := range anchorIndexes {
		start := max(0, anchor-bufferBefore)
		end := min(len(ordered)-1, anchor+bufferAfter)
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
		result = append(result, cloneEvent(ordered[i]))
	}
	return result
}

func validateQueryVector(queryVector []float64) (float64, error) {
	if len(queryVector) == 0 {
		return 0, errRetrieveQueryVectorRequired
	}

	normSquared := 0.0
	for _, value := range queryVector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, errRetrieveQueryVectorNonFinite
		}
		normSquared += value * value
	}
	if normSquared == 0 {
		return 0, errRetrieveQueryVectorZeroNorm
	}

	return math.Sqrt(normSquared), nil
}

func vectorNorm(values []float64) float64 {
	normSquared := 0.0
	for _, value := range values {
		normSquared += value * value
	}
	return math.Sqrt(normSquared)
}

func dotProduct(left, right []float64) float64 {
	result := 0.0
	for i := range left {
		result += left[i] * right[i]
	}
	return result
}

func cloneEvent(event Event) Event {
	cloned := event
	cloned.RepresentativeKeys = deepCopyRepresentativeKeys(event.RepresentativeKeys)
	return cloned
}

func deepCopyRepresentativeKeys(representativeKeys [][]float64) [][]float64 {
	if representativeKeys == nil {
		return nil
	}

	cloned := make([][]float64, len(representativeKeys))
	for i, key := range representativeKeys {
		cloned[i] = append([]float64(nil), key...)
	}
	return cloned
}
