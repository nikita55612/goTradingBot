package trading

import (
	"sync/atomic"

	"github.com/nikita55612/goTradingBot/internal/cdl"
)

type Instrument struct {
	symbol   string
	interval cdl.Interval

	minOrderAmt float64
	tickSize    float64
}

type Strategy struct {
	lastPrice       atomic.Pointer[float64]
	limitCeilPrice  atomic.Pointer[float64]
	limitFloorPrice atomic.Pointer[float64]
}

func (s *Strategy) background() {
}
