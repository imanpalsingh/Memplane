package memory

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestRefineBoundariesByModularityRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := RefineBoundariesByModularity([]int{2}, nil, 1)
	if !errors.Is(err, errRefineSimilarityEmpty) {
		t.Fatalf("expected error %v, got %v", errRefineSimilarityEmpty, err)
	}

	_, err = RefineBoundariesByModularity([]int{2}, [][]float64{{1, 0}, {0}}, 1)
	if !errors.Is(err, errRefineSimilarityNotSquare) {
		t.Fatalf("expected error %v, got %v", errRefineSimilarityNotSquare, err)
	}

	_, err = RefineBoundariesByModularity([]int{2}, [][]float64{{1, 0.2}, {0.1, 1}}, 1)
	if !errors.Is(err, errRefineSimilarityAsymmetric) {
		t.Fatalf("expected error %v, got %v", errRefineSimilarityAsymmetric, err)
	}

	_, err = RefineBoundariesByModularity([]int{2}, [][]float64{{1, -0.1}, {-0.1, 1}}, 1)
	if !errors.Is(err, errRefineSimilarityNegative) {
		t.Fatalf("expected error %v, got %v", errRefineSimilarityNegative, err)
	}

	_, err = RefineBoundariesByModularity([]int{2}, [][]float64{{1, math.NaN()}, {math.NaN(), 1}}, 1)
	if !errors.Is(err, errRefineSimilarityNonFinite) {
		t.Fatalf("expected error %v, got %v", errRefineSimilarityNonFinite, err)
	}
}

func TestRefineBoundariesByModularityShiftsBoundary(t *testing.T) {
	t.Parallel()

	// Best split should be at boundary 2 for this affinity structure.
	keySimilarity := [][]float64{
		{1, 4, 0.1, 0.1, 0.1, 0.1},
		{4, 1, 0.1, 0.1, 0.1, 0.1},
		{0.1, 0.1, 1, 4, 4, 4},
		{0.1, 0.1, 4, 1, 4, 4},
		{0.1, 0.1, 4, 4, 1, 4},
		{0.1, 0.1, 4, 4, 4, 1},
	}

	refined, err := RefineBoundariesByModularity([]int{3}, keySimilarity, 1)
	if err != nil {
		t.Fatalf("refine boundaries: %v", err)
	}

	want := []int{2}
	if !reflect.DeepEqual(refined, want) {
		t.Fatalf("expected refined boundaries %v, got %v", want, refined)
	}
}
