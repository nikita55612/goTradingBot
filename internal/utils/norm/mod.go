package norm

import (
	"math"
	"slices"

	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func SequenceDiff[V Number](s1 []V, s2 []V, shift int) []float64 {
	n, m := len(s1), len(s2)
	if n == 0 || m == 0 || n-m != 0 {
		panic("SequenceDiff: slice mismatch")
	}
	res := make([]float64, n)
	if shift == 0 {
		return res
	}
	for i := shift; i < n; i++ {
		res[i] = float64(s1[i]) - float64(s2[i-shift])
	}
	return res
}

func ZScoreNormalize[V Number](s []V, period int) []float64 {
	n := len(s)
	if n < 2 {
		panic("ZScoreNormalize: slice length is less than 2")
	}
	normalized := make([]float64, n)
	for i := 0; i < n; i++ {
		window := s[max(0, i-period+1) : i+1]
		mean := numeric.Avg(window)
		var sumSqr float64
		for _, v := range window {
			diff := float64(v) - mean
			sumSqr += diff * diff
		}
		variance := sumSqr / float64(len(window))
		if variance < 0 {
			variance = 0
		}
		stdDev := math.Sqrt(variance)
		if stdDev == 0 {
			normalized[i] = 0
		} else {
			normalized[i] = (float64(s[i]) - mean) / stdDev
		}
	}
	return normalized
}

func MinusOneOneNormalize[V Number](s []V, period int) []float64 {
	n := len(s)
	if n < 2 {
		panic("MinusOneOneNormalize: slice length is less than 2")
	}
	normalized := make([]float64, n)
	for i := 0; i < n; i++ {
		window := s[max(0, i-period+1) : i+1]
		v := float64(s[i])
		minV := float64(slices.Min(window))
		maxV := float64(slices.Max(window))
		if minV == maxV {
			normalized[i] = 0
		} else {
			normalized[i] = 2*((v-minV)/(maxV-minV)) - 1
		}
	}
	return normalized
}

func ZeroOneNormalize[V Number](s []V, period int) []float64 {
	n := len(s)
	if n < 2 {
		panic("ZeroOneNormalize: slice length is less than 2")
	}
	normalized := make([]float64, n)
	for i := 0; i < n; i++ {
		window := s[max(0, i-period+1) : i+1]
		v := float64(s[i])
		minV := float64(slices.Min(window))
		maxV := float64(slices.Max(window))
		if minV == maxV {
			normalized[i] = 0
		} else {
			normalized[i] = (v - minV) / (maxV - minV)
		}
	}
	return normalized
}
