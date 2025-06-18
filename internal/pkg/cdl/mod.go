package cdl

import (
	"strconv"
)

type Candle struct {
	Time     int64
	O        float64
	H        float64
	L        float64
	C        float64
	Volume   float64
	Turnover float64
}

type CandleStreamData struct {
	Candle   Candle
	Interval Interval
	Confirm  bool
}

func (c *Candle) AsArr() *[7]string {
	return &[7]string{
		strconv.FormatInt(c.Time, 10),
		strconv.FormatFloat(c.O, 'f', -1, 64),
		strconv.FormatFloat(c.H, 'f', -1, 64),
		strconv.FormatFloat(c.L, 'f', -1, 64),
		strconv.FormatFloat(c.C, 'f', -1, 64),
		strconv.FormatFloat(c.Volume, 'f', -1, 64),
		strconv.FormatFloat(c.Turnover, 'f', -1, 64),
	}
}
