package memory

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestBuildEventsFromSurpriseBuildsContiguousEvents(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	surprise := []float64{0.05, 0.2, 1.2, 0.1, 0.15, 1.5, 0.2}

	events, boundaries, err := BuildEventsFromSurprise(
		"tenant_1",
		"session_1",
		100,
		surprise,
		identitySimilarity(len(surprise)),
		0.8,
		1,
		createdAt,
		"seg",
	)
	if err != nil {
		t.Fatalf("build events: %v", err)
	}

	wantBoundaries := []int{103, 105}
	if !reflect.DeepEqual(boundaries, wantBoundaries) {
		t.Fatalf("expected boundaries %v, got %v", wantBoundaries, boundaries)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].EventID != "seg_0" || events[0].StartToken != 100 || events[0].EndTokenExclusive != 103 {
		t.Fatalf("unexpected first event: %#v", events[0])
	}
	if events[1].EventID != "seg_1" || events[1].StartToken != 103 || events[1].EndTokenExclusive != 105 {
		t.Fatalf("unexpected second event: %#v", events[1])
	}
	if events[2].EventID != "seg_2" || events[2].StartToken != 105 || events[2].EndTokenExclusive != 107 {
		t.Fatalf("unexpected third event: %#v", events[2])
	}
}

func TestBuildEventsFromSurpriseRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	validSurprise := []float64{0.1, 1.0, 0.1}

	cases := []struct {
		name string
		err  error
		run  func() error
	}{
		{
			name: "negative start token",
			err:  errSegmentStartTokenNegative,
			run: func() error {
				_, _, err := BuildEventsFromSurprise("tenant_1", "session_1", -1, validSurprise, identitySimilarity(len(validSurprise)), 0.5, 1, createdAt, "seg")
				return err
			},
		},
		{
			name: "missing surprise",
			err:  errSegmentSurpriseRequired,
			run: func() error {
				_, _, err := BuildEventsFromSurprise("tenant_1", "session_1", 0, nil, identitySimilarity(len(validSurprise)), 0.5, 1, createdAt, "seg")
				return err
			},
		},
		{
			name: "missing key similarity",
			err:  errSegmentKeySimilarityRequired,
			run: func() error {
				_, _, err := BuildEventsFromSurprise("tenant_1", "session_1", 0, validSurprise, nil, 0.5, 1, createdAt, "seg")
				return err
			},
		},
		{
			name: "missing event prefix",
			err:  errSegmentEventIDPrefixMissing,
			run: func() error {
				_, _, err := BuildEventsFromSurprise("tenant_1", "session_1", 0, validSurprise, identitySimilarity(len(validSurprise)), 0.5, 1, createdAt, "")
				return err
			},
		},
		{
			name: "invalid threshold from detector",
			err:  errNegativeSurpriseThreshold,
			run: func() error {
				_, _, err := BuildEventsFromSurprise("tenant_1", "session_1", 0, validSurprise, identitySimilarity(len(validSurprise)), -0.1, 1, createdAt, "seg")
				return err
			},
		},
		{
			name: "mismatched similarity length",
			err:  errSegmentKeySimilarityLength,
			run: func() error {
				_, _, err := BuildEventsFromSurprise(
					"tenant_1",
					"session_1",
					0,
					validSurprise,
					[][]float64{{1}},
					0.5,
					1,
					createdAt,
					"seg",
				)
				return err
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.run()
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected error %v, got %v", tc.err, err)
			}
		})
	}
}

func identitySimilarity(size int) [][]float64 {
	similarity := make([][]float64, size)
	for i := 0; i < size; i++ {
		row := make([]float64, size)
		row[i] = 1
		similarity[i] = row
	}
	return similarity
}
