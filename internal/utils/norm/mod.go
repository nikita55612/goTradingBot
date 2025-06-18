package norm

import (
	"math"

	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func ZScore[V Number](s []V) float64 {
	n := len(s)
	if n <= 1 {
		return 0
	}
	mean := numeric.Avg(s)
	var sumSqr float64
	for _, v := range s {
		diff := float64(v) - mean
		sumSqr += diff * diff
	}
	variance := sumSqr / float64(n)
	if variance < 0 {
		variance = 0
	}
	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	} else {
		return (float64(s[n-1]) - mean) / stdDev
	}
}

func ZScoreNormalize[V Number](s []V, period int) []float64 {
	n := len(s)
	if n < period {
		panic("slice length is less than period")
	}
	normalized := make([]float64, n)
	for i := 0; i < n; i++ {
		window := s[max(0, i-period+1) : i+1]
		normalized[i] = ZScore(window)
	}
	return normalized
}
