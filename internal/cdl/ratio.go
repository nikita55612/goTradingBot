package cdl

import "math"

// CandleRatio - тип для выбора соотношения между свечами
type CandleRatio string

// Константы соотношений между свечами
const (
	BodyStrengthRatio  CandleRatio = "AMR" // Соотношение размеров тел (текущее/предыдущее)
	LowerWickRatio     CandleRatio = "LWR" // Соотношение нижних теней
	UpperWickRatio     CandleRatio = "UWR" // Соотношение верхних теней
	ClosePositionRatio CandleRatio = "CPR" // Положение Close в диапазоне предыдущей свечи
	MomentumRatio      CandleRatio = "MR"  // Соотношение моментумов
	BreakoutPower      CandleRatio = "BP"  // Сила пробоя относительно предыдущего диапазона
	VolumeRatio        CandleRatio = "VR"  // Соотношение объемов
	TrueRangeRatio     CandleRatio = "TRR" // Истинный диапазон (max(H-L, |H-PrevC|, |L-PrevC|))
)

// ListOfCandleRatio возвращает список соотношений между свечами с заданным сдвигом
func ListOfCandleRatio(candles []Candle, r CandleRatio, shift int) []float64 {
	if shift == 0 {
		panic("shift не может быть 0")
	}

	ratios := make([]float64, len(candles))
	for i := shift; i < len(candles); i++ {
		ratios[i] = candles[i].Ratio(r, &candles[i-shift])
	}
	return ratios
}

// Ratio вычисляет соотношение между текущей и предыдущей свечой
func (c *Candle) Ratio(r CandleRatio, pc *Candle) float64 {
	if pc == nil {
		return 0
	}

	switch r {
	case BodyStrengthRatio:
		if prev := pc.Arg(Body); prev != 0 {
			return c.Arg(Body) / prev
		}
	case LowerWickRatio:
		if prev := pc.Arg(LowerWick); prev != 0 {
			return c.Arg(LowerWick) / prev
		}
	case UpperWickRatio:
		if prev := pc.Arg(UpperWick); prev != 0 {
			return c.Arg(UpperWick) / prev
		}
	case ClosePositionRatio:
		if prevTr := pc.Arg(TrueRange); prevTr != 0 {
			return (c.C - pc.L) / prevTr
		}
	case MomentumRatio:
		if prev := pc.Arg(Momentum); prev != 0 {
			return c.Arg(Momentum) / prev
		}
	case BreakoutPower:
		if prevTr := pc.Arg(TrueRange); prevTr != 0 {
			if c.C > pc.H {
				return (c.C - pc.H) / prevTr
			}
			if c.C <= pc.L {
				return (pc.L - c.C) / prevTr
			}
		}
	case VolumeRatio:
		if pc.Volume != 0 {
			return c.Volume / pc.Volume
		}
		return 1
	case TrueRangeRatio:
		return max(c.Arg(TrueRange), math.Abs(c.H-pc.C), math.Abs(c.L-pc.C))
	}
	return 0
}
