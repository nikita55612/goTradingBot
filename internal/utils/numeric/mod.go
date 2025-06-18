package numeric

import (
	"math"
	"slices"
	"sort"
	"strconv"

	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

// Sum вычисляет сумму всех элементов в слайсе.
func Sum[V Number](s []V) V {
	var res V
	for _, v := range s {
		res += v
	}
	return res
}

// Avg вычисляет среднее арифметическое значение элементов в слайсе.
// Использует устойчивый алгоритм вычисления среднего.
// Может накапливаться ошибка округления для больших массивов.
func Avg[V Number](s []V) float64 {
	if len(s) == 0 {
		return 0
	}
	var mean float64
	for i, v := range s {
		mean += (float64(v) - mean) / float64(i+1)
	}
	return mean
}

// Median вычисляет медианное значение элементов в слайсе.
func Median[V Number](s []V) float64 {
	n := len(s)
	if n == 0 {
		return 0
	}
	values := make([]float64, n)
	for i, v := range s {
		values[i] = float64(v)
	}
	sort.Float64s(values)
	if n%2 == 1 {
		return values[n/2]
	}
	return SafeAvg2Val(values[n/2-1], values[n/2])
}

// Quantile вычисляет квантиль для заданной позиции 0.0-1.0
func Quantile[V Number](s []V, pos float64) float64 {
	if len(s) == 0 {
		return 0
	}
	if pos <= 0 {
		return float64(slices.Min(s))
	}
	if pos >= 1 {
		return float64(slices.Max(s))
	}
	values := make([]float64, len(s))
	for i, v := range s {
		values[i] = float64(v)
	}
	sort.Float64s(values)
	if len(values) == 1 {
		return values[0]
	}
	exactPos := pos * float64(len(values)-1)
	lower := int(exactPos)
	weight := exactPos - float64(lower)
	if lower+1 >= len(values) {
		return values[lower]
	}
	return values[lower] + weight*(values[lower+1]-values[lower])
}

// SafeAvg2Val вычисляет среднее между двумя значениями.
// Использует устойчивую формулу вычисления среднего для больших чисел.
func SafeAvg2Val[V Number](a, b V) float64 {
	return float64(a) + (float64(b)-float64(a))/2
}

// DiffPercent вычисляет процентное изменение между двумя значениями.
func DiffPercent[V Number](a, b V) float64 {
	if a == 0 {
		if b > 0 {
			return math.Inf(1)
		} else if b < 0 {
			return math.Inf(-1)
		}
		return math.NaN()
	}
	return (float64(b) - float64(a)) / float64(a) * 100
}

// CalculateSlopeAngle вычисляет угол наклона линии тренда в градусах.
// mult - множитель для значений по оси X (используется для масштабирования).
func CalculateSlopeAngle(y []float64, mult float64) float64 {
	n := float64(len(y))
	var sumX, sumY, sumXY, sumXX float64
	for i, yi := range y {
		xi := float64(i) * mult
		sumX += xi
		sumY += yi
		sumXY += xi * yi
		sumXX += xi * xi
	}
	numerator := n*sumXY - sumX*sumY
	denominator := n*sumXX - sumX*sumX
	if denominator == 0 {
		return 0
	}
	slope := numerator / denominator
	return math.Atan(slope) * 180 / math.Pi
}

// Transpose возвращает транспонированную версию матрицы.
// Пример: [[1,2],[3,4]] -> [[1,3],[2,4]]
func TransposeMatrix[V Number](m [][]V) [][]V {
	if len(m) == 0 {
		panic("empty rows")
	}
	rows, cols := len(m), len(m[0])
	result := make([][]V, cols)
	for i := range result {
		result[i] = make([]V, rows)
		for j := 0; j < rows; j++ {
			result[i][j] = m[j][i]
		}
	}
	return result
}

// DecimalPlaces возвращает количество знаков после запятой в числе float64.
func DecimalPlaces(f float64) int {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return len(s) - i - 1
		}
	}
	return 0
}

// TruncateFloat усекает float64 до precision знаков после запятой (без округления).
func TruncateFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Trunc(val*ratio) / ratio
}

// RoundFloat округляет float64 до precision знаков после запятой.
func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// CeilFloat округляет float64 вверх (к потолку) до precision знаков после запятой.
func CeilFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Ceil(val*ratio) / ratio
}

// FloorFloat округляет float64 вниз (к полу) до precision знаков после запятой.
func FloorFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Floor(val*ratio) / ratio
}
