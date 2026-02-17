package memory

import (
	"errors"
	"fmt"
	"time"
)

var (
	errSegmentStartTokenNegative    = errors.New("start_token must be non-negative")
	errSegmentSurpriseRequired      = errors.New("surprise must contain at least one value")
	errSegmentKeySimilarityRequired = errors.New("key_similarity is required")
	errSegmentEventIDPrefixMissing  = errors.New("event id prefix is required")
	errInvalidBoundary              = errors.New("detected boundary is outside valid token range")
	errSegmentKeySimilarityLength   = errors.New("key_similarity size must match surprise length")
)

func BuildEventsFromSurprise(
	tenantID string,
	sessionID string,
	startToken int,
	surprise []float64,
	keySimilarity [][]float64,
	threshold float64,
	minBoundaryGap int,
	createdAt time.Time,
	eventIDPrefix string,
) ([]Event, []int, error) {
	if startToken < 0 {
		return nil, nil, errSegmentStartTokenNegative
	}
	if len(surprise) == 0 {
		return nil, nil, errSegmentSurpriseRequired
	}
	if len(keySimilarity) == 0 {
		return nil, nil, errSegmentKeySimilarityRequired
	}
	if eventIDPrefix == "" {
		return nil, nil, errSegmentEventIDPrefixMissing
	}
	if len(keySimilarity) != len(surprise) {
		return nil, nil, errSegmentKeySimilarityLength
	}

	boundariesRelative, err := DetectBoundaries(surprise, threshold, minBoundaryGap)
	if err != nil {
		return nil, nil, err
	}
	boundariesRelative, err = RefineBoundariesByModularity(boundariesRelative, keySimilarity, minBoundaryGap)
	if err != nil {
		return nil, nil, err
	}

	endToken := startToken + len(surprise)
	boundaries := make([]int, len(boundariesRelative))
	for i, boundary := range boundariesRelative {
		absoluteBoundary := startToken + boundary
		if absoluteBoundary <= startToken || absoluteBoundary >= endToken {
			return nil, nil, errInvalidBoundary
		}
		boundaries[i] = absoluteBoundary
	}

	events := make([]Event, 0, len(boundaries)+1)
	cursor := startToken
	for i, boundary := range boundaries {
		event, err := NewEvent(
			fmt.Sprintf("%s_%d", eventIDPrefix, i),
			tenantID,
			sessionID,
			cursor,
			boundary,
			createdAt,
		)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, event)
		cursor = boundary
	}

	event, err := NewEvent(
		fmt.Sprintf("%s_%d", eventIDPrefix, len(boundaries)),
		tenantID,
		sessionID,
		cursor,
		endToken,
		createdAt,
	)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, event)

	return events, boundaries, nil
}
