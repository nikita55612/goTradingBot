package trading

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/nikita55612/goTradingBot/internal/cdl"
	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
	"github.com/nikita55612/goTradingBot/internal/utils/seqs"
)

type Instrument struct {
	symbol      string
	interval    cdl.Interval
	minOrderAmt float64
	tickSize    float64
}

type Strategy struct {
	symbol            string
	interval          cdl.Interval
	qtyPrecision      int
	minOrderAmt       float64
	tickSize          float64
	tickSizePrecision int

	ctx          context.Context
	subData      *SubData
	orderRequest chan<- *OrderRequest

	candleStreamChan chan *cdl.CandleStreamData
	candleStream     chan<- struct{}

	confirmHandlerChan chan *cdl.Candle
	backgroundChan     chan *cdl.Candle

	balance   float64
	longRatio float64

	orderLog *seqs.OrderedMap[string, *Order]

	limitOrderOffset float64

	lastPrice       atomic.Pointer[float64]
	limitCeilPrice  atomic.Pointer[float64]
	limitFloorPrice atomic.Pointer[float64]
}

func NewStrategy(
	symbol string,
	interval cdl.Interval,
	balance float64,
	longRatio float64,
) *Strategy {
	return &Strategy{}
}

func (s *Strategy) Init(ctx context.Context, subData *SubData, req chan<- *OrderRequest) {
	s.ctx = ctx
	s.subData = subData
	s.orderRequest = req
}

func (s *Strategy) Start() error {
	instrumentInfo, err := s.subData.GetInstrumentInfo(s.symbol)
	if err != nil {
		return err
	}
	s.qtyPrecision = instrumentInfo.QtyPrecision
	s.minOrderAmt = instrumentInfo.MinOrderAmt
	s.tickSize = instrumentInfo.TickSize
	s.tickSizePrecision = numeric.DecimalPlaces(s.tickSize)

	lastCandle, err := s.subData.GetCandles(s.symbol, s.interval, 1)
	if err != nil || len(lastCandle) == 0 {
		return err
	}
	s.lastPrice.Store(&lastCandle[0].C)
	s.limitCeilPrice.Store(&lastCandle[0].C)
	s.limitFloorPrice.Store(&lastCandle[0].C)

	s.candleStreamChan = make(chan *cdl.CandleStreamData)
	done, err := s.subData.SubscribeChan(s.symbol, s.interval, s.candleStreamChan)
	if err != nil {
		close(s.candleStreamChan)
		return err
	}
	s.candleStream = done
	s.confirmHandlerChan = make(chan *cdl.Candle)
	s.backgroundChan = make(chan *cdl.Candle)

	go s.background()
	go s.confirmHandler()
	go s.observe()
	go func() {
		<-s.ctx.Done()
		s.Stop()
	}()

	return nil
}

func (s *Strategy) Stop() {
	close(s.candleStream)
	close(s.confirmHandlerChan)
	close(s.backgroundChan)
}

func (s *Strategy) observe() {
	for data := range s.candleStreamChan {
		s.backgroundChan <- &data.Candle
		if data.Confirm {
			s.confirmHandlerChan <- &data.Candle
		}
	}
}

func (s *Strategy) background() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				lastPrice := *s.lastPrice.Load()
				limitCeilPrice := numeric.TruncateFloat(
					lastPrice*(1+s.limitOrderOffset), s.tickSizePrecision,
				)
				s.limitCeilPrice.Store(&limitCeilPrice)
				limitFloorPrice := numeric.TruncateFloat(
					lastPrice*(1-s.limitOrderOffset), s.tickSizePrecision,
				)
				s.limitFloorPrice.Store(&limitFloorPrice)
			}
		}
	}()
	for candle := range s.backgroundChan {
		s.lastPrice.Store(&candle.C)
	}
}

func (s *Strategy) confirmHandler() {
	for candle := range s.confirmHandlerChan {

	}
}
