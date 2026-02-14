package memory

import (
	"errors"
	"reflect"
	"testing"
)

func TestDetectBoundariesRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := DetectBoundaries([]float64{0, 1, 0}, -0.1, 1)
	if !errors.Is(err, errNegativeSurpriseThreshold) {
		t.Fatalf("expected error %v, got %v", errNegativeSurpriseThreshold, err)
	}

	_, err = DetectBoundaries([]float64{0, 1, 0}, 0.5, 0)
	if !errors.Is(err, errInvalidMinBoundaryGap) {
		t.Fatalf("expected error %v, got %v", errInvalidMinBoundaryGap, err)
	}
}

func TestDetectBoundariesHandlesShortInput(t *testing.T) {
	t.Parallel()

	for _, scores := range [][]float64{{}, {0.5}, {0.1, 0.9}} {
		got, err := DetectBoundaries(scores, 0.8, 1)
		if err != nil {
			t.Fatalf("detect boundaries: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no boundaries for %v, got %v", scores, got)
		}
	}
}

func TestDetectBoundariesDetectsSurprisePeaks(t *testing.T) {
	t.Parallel()

	scores := []float64{0.05, 0.2, 1.2, 0.1, 0.15, 1.5, 0.2}
	got, err := DetectBoundaries(scores, 0.8, 1)
	if err != nil {
		t.Fatalf("detect boundaries: %v", err)
	}

	want := []int{3, 6}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected boundaries %v, got %v", want, got)
	}
}

func TestDetectBoundariesThresholdIsStrictlyGreater(t *testing.T) {
	t.Parallel()

	scores := []float64{0.1, 0.5, 0.1}
	got, err := DetectBoundaries(scores, 0.5, 1)
	if err != nil {
		t.Fatalf("detect boundaries: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected no boundaries, got %v", got)
	}
}

func TestDetectBoundariesPrefersStrongerPeakWithinGap(t *testing.T) {
	t.Parallel()

	scores := []float64{0.1, 1.1, 0.2, 1.5, 0.1}
	got, err := DetectBoundaries(scores, 0.8, 3)
	if err != nil {
		t.Fatalf("detect boundaries: %v", err)
	}

	want := []int{4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected boundaries %v, got %v", want, got)
	}
}

func TestDetectBoundariesRejectsPlateauAsPeak(t *testing.T) {
	t.Parallel()

	scores := []float64{0.1, 1.0, 1.0, 0.1}
	got, err := DetectBoundaries(scores, 0.8, 1)
	if err != nil {
		t.Fatalf("detect boundaries: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected no boundaries, got %v", got)
	}
}
