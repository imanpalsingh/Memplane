package memory

import "errors"

var (
	errNegativeSurpriseThreshold = errors.New("surprise threshold must be non-negative")
	errInvalidMinBoundaryGap     = errors.New("minimum boundary gap must be positive")
)

func DetectBoundaries(surprise []float64, threshold float64, minBoundaryGap int) ([]int, error) {
	if threshold < 0 {
		return nil, errNegativeSurpriseThreshold
	}
	if minBoundaryGap <= 0 {
		return nil, errInvalidMinBoundaryGap
	}
	if len(surprise) < 3 {
		return []int{}, nil
	}

	boundaries := make([]int, 0)
	peaks := make([]float64, 0)

	for i := 1; i < len(surprise)-1; i++ {
		score := surprise[i]
		if score <= threshold {
			continue
		}
		// Accept only local maxima so boundaries represent transition peaks.
		if score <= surprise[i-1] || score <= surprise[i+1] {
			continue
		}

		// Boundary is the first token after the peak token.
		boundary := i + 1
		last := len(boundaries) - 1
		if last < 0 {
			boundaries = append(boundaries, boundary)
			peaks = append(peaks, score)
			continue
		}

		if boundary-boundaries[last] >= minBoundaryGap {
			boundaries = append(boundaries, boundary)
			peaks = append(peaks, score)
			continue
		}

		// If two peaks are too close, keep the stronger one.
		if score > peaks[last] {
			boundaries[last] = boundary
			peaks[last] = score
		}
	}

	return boundaries, nil
}
