package cdl

import (
	"math"
)

// CandleArg - тип для выбора параметра свечи
type CandleArg string

// Константы параметров свечи
const (
	Time               CandleArg = "T"        // Время открытия (timestamp)
	Open               CandleArg = "O"        // Цена открытия
	High               CandleArg = "H"        // Максимальная цена
	Low                CandleArg = "L"        // Минимальная цена
	Close              CandleArg = "C"        // Цена закрытия
	CL                 CandleArg = "CL"       // Среднее Close и Low
	CH                 CandleArg = "CH"       // Среднее Close и High
	HL                 CandleArg = "HL"       // Среднее High и Low
	HLC                CandleArg = "HLC"      // Типичная цена (High+Low+Close)/3
	OHLC               CandleArg = "OHLC"     // Среднее всех цен (O+H+L+C)/4
	HLCC               CandleArg = "HLCC"     // Взвешенное Close (H+L+2C)/4
	Volume             CandleArg = "V"        // Объем торгов
	Turnover           CandleArg = "Turnover" // Оборот
	TrueRange          CandleArg = "TR"       // Диапазон High-Low
	NormalizedRange    CandleArg = "NR"       // Норм. диапазон (H-L)/Open
	RateOfChange       CandleArg = "ROC"      // Изменение цены (C-O)/O
	Momentum           CandleArg = "M"        // Импульс C-O
	Acceleration       CandleArg = "Acc"      // Ускорение (C-O)/(H-L)
	PriceVolume        CandleArg = "PV"       // Цена*Объем (C*V)
	Body               CandleArg = "AM"       // Тело свечи |C-O|
	UpperWick          CandleArg = "UW"       // Верхняя тень H-Max(O,C)
	LowerWick          CandleArg = "LW"       // Нижняя тень Min(O,C)-L
	WickRatio          CandleArg = "WR"       // Соотношение теней UW/LW
	BodyRangeRatio     CandleArg = "AMRR"     // Тело/Диапазон |C-O|/(H-L)
	Direction          CandleArg = "Dir"      // Направление: 1(бычья), -1(медвежья), 0(доджи)
	WeightedClose      CandleArg = "OHLCC"    // Взвешенное закрытие (O+H+L+2C)/5
	VWAP               CandleArg = "VWAP"     // Средневзвешенная цена (Turnover/Volume)
	CloseLocationValue CandleArg = "CLV"      // Положение Close в диапазоне (C-L)/(H-L)
	ShadowRatio        CandleArg = "SR"       // Соотношение теней к телу (UW+LW)/Body
)

// ListOfCandleArg возвращает список значений указанного параметра свечей
func ListOfCandleArg(candles []Candle, arg CandleArg) []float64 {
	list := make([]float64, len(candles))

	for i := range candles {
		list[i] = candles[i].Arg(arg)
	}

	return list
}

// Arg возвращает значение указанного параметра свечи
func (c *Candle) Arg(a CandleArg) float64 {
	switch a {
	case Time:
		return float64(c.Time)
	case Open:
		return c.O
	case High:
		return c.H
	case Low:
		return c.L
	case Close:
		return c.C
	case CL:
		return (c.C + c.L) / 2
	case CH:
		return (c.C + c.H) / 2
	case HL:
		return (c.H + c.L) / 2
	case HLC:
		return (c.H + c.L + c.C) / 3
	case OHLC:
		return (c.O + c.H + c.L + c.C) / 4
	case HLCC:
		return (c.H + c.L + 2*c.C) / 4
	case TrueRange:
		return c.H - c.L
	case Momentum:
		return c.C - c.O
	case Acceleration:
		if c.H != c.L {
			return (c.C - c.O) / (c.H - c.L)
		}
	case NormalizedRange:
		if c.O != 0 {
			return (c.H - c.L) / c.O
		}
	case RateOfChange:
		if c.O != 0 {
			return (c.C - c.O) / c.O
		}
	case Volume:
		return c.Volume
	case PriceVolume:
		return c.Volume * c.C
	case Turnover:
		return c.Turnover
	case Body:
		return math.Abs(c.C - c.O)
	case UpperWick:
		return c.H - max(c.O, c.C)
	case LowerWick:
		return min(c.O, c.C) - c.L
	case BodyRangeRatio:
		if c.H != c.L {
			return math.Abs(c.C-c.O) / (c.H - c.L)
		}
	case Direction:
		if c.C > c.O {
			return 1
		} else if c.C < c.O {
			return -1
		}
		return 0
	case WeightedClose:
		return (c.O + c.H + c.L + 2*c.C) / 5
	case VWAP:
		if c.Volume != 0 {
			return c.Turnover / c.Volume
		}
	case CloseLocationValue:
		if tr := c.H - c.L; tr != 0 {
			return (c.C - c.L) / tr
		}
	case ShadowRatio:
		if body := math.Abs(c.C - c.O); body != 0 {
			return (c.H - c.L - body) / body
		}
	}
	return 0
}
