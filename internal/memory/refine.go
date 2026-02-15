package memory

import (
	"errors"
	"math"
)

var (
	errRefineSimilarityEmpty          = errors.New("key_similarity is required")
	errRefineSimilarityNotSquare      = errors.New("key_similarity must be a square matrix")
	errRefineSimilarityAsymmetric     = errors.New("key_similarity must be symmetric")
	errRefineSimilarityNegative       = errors.New("key_similarity must have non-negative weights")
	errRefineSimilarityNonFinite      = errors.New("key_similarity must not contain NaN or Inf")
	errRefineBoundaryOutOfRange       = errors.New("boundary is outside valid token range")
	errRefineBoundaryOrderInvalid     = errors.New("boundaries must be strictly increasing")
	errRefineBoundaryGapInvalid       = errors.New("boundaries violate minimum boundary gap")
	errRefineNonPositiveTotalAffinity = errors.New("key_similarity has non-positive total affinity")
)

const similarityTolerance = 1e-9

func RefineBoundariesByModularity(
	initialBoundaries []int,
	keySimilarity [][]float64,
	minBoundaryGap int,
) ([]int, error) {
	if minBoundaryGap <= 0 {
		return nil, errInvalidMinBoundaryGap
	}
	if len(keySimilarity) == 0 {
		return nil, errRefineSimilarityEmpty
	}
	tokenCount := len(keySimilarity)
	for _, row := range keySimilarity {
		if len(row) != tokenCount {
			return nil, errRefineSimilarityNotSquare
		}
	}
	if err := validateSimilarityMatrix(keySimilarity); err != nil {
		return nil, err
	}
	if len(initialBoundaries) == 0 {
		return []int{}, nil
	}
	if err := validateBoundaries(initialBoundaries, tokenCount, minBoundaryGap); err != nil {
		return nil, err
	}

	refined := make([]int, 0, len(initialBoundaries))
	for i, boundary := range initialBoundaries {
		alpha := 0
		if i > 0 {
			alpha = refined[i-1]
		}
		beta := tokenCount
		if i+1 < len(initialBoundaries) {
			beta = initialBoundaries[i+1]
		}

		candidateMin := alpha + minBoundaryGap
		candidateMax := beta - minBoundaryGap
		if candidateMin > candidateMax {
			return nil, errRefineBoundaryGapInvalid
		}

		bestBoundary := boundary
		if bestBoundary < candidateMin {
			bestBoundary = candidateMin
		}
		if bestBoundary > candidateMax {
			bestBoundary = candidateMax
		}

		stats, err := buildIntervalStats(keySimilarity, alpha, beta)
		if err != nil {
			return nil, err
		}

		bestScore, err := splitModularityScore(keySimilarity, stats, bestBoundary)
		if err != nil {
			return nil, err
		}

		for candidate := candidateMin; candidate <= candidateMax; candidate++ {
			score, err := splitModularityScore(keySimilarity, stats, candidate)
			if err != nil {
				return nil, err
			}
			if score > bestScore {
				bestScore = score
				bestBoundary = candidate
			}
		}

		refined = append(refined, bestBoundary)
	}

	return refined, nil
}

func validateBoundaries(boundaries []int, tokenCount, minBoundaryGap int) error {
	prev := 0
	for i, boundary := range boundaries {
		if boundary <= 0 || boundary >= tokenCount {
			return errRefineBoundaryOutOfRange
		}
		if i > 0 {
			if boundary <= prev {
				return errRefineBoundaryOrderInvalid
			}
			if boundary-prev < minBoundaryGap {
				return errRefineBoundaryGapInvalid
			}
		}
		prev = boundary
	}
	return nil
}

func validateSimilarityMatrix(keySimilarity [][]float64) error {
	for i, row := range keySimilarity {
		for j := i; j < len(row); j++ {
			left := row[j]
			right := keySimilarity[j][i]

			if math.IsNaN(left) || math.IsInf(left, 0) || math.IsNaN(right) || math.IsInf(right, 0) {
				return errRefineSimilarityNonFinite
			}
			if left < -similarityTolerance || right < -similarityTolerance {
				return errRefineSimilarityNegative
			}
			if math.Abs(left-right) > similarityTolerance {
				return errRefineSimilarityAsymmetric
			}
		}
	}
	return nil
}

type intervalStats struct {
	alpha         int
	beta          int
	degree        []float64
	totalAffinity float64
}

func buildIntervalStats(keySimilarity [][]float64, alpha, beta int) (intervalStats, error) {
	degree := make([]float64, beta-alpha)
	totalAffinity := 0.0
	for i := alpha; i < beta; i++ {
		for j := alpha; j < beta; j++ {
			w := keySimilarity[i][j]
			degree[i-alpha] += w
			totalAffinity += w
		}
	}
	if totalAffinity <= 0 {
		return intervalStats{}, errRefineNonPositiveTotalAffinity
	}

	return intervalStats{
		alpha:         alpha,
		beta:          beta,
		degree:        degree,
		totalAffinity: totalAffinity,
	}, nil
}

func splitModularityScore(keySimilarity [][]float64, stats intervalStats, split int) (float64, error) {
	if split <= stats.alpha || split >= stats.beta {
		return math.Inf(-1), nil
	}

	modularity := 0.0
	for i := stats.alpha; i < stats.beta; i++ {
		for j := stats.alpha; j < stats.beta; j++ {
			sameCluster := (i < split && j < split) || (i >= split && j >= split)
			if !sameCluster {
				continue
			}
			expected := (stats.degree[i-stats.alpha] * stats.degree[j-stats.alpha]) / stats.totalAffinity
			modularity += keySimilarity[i][j] - expected
		}
	}

	return modularity / stats.totalAffinity, nil
}
