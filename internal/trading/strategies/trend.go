package strategies

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/trading"
	"github.com/nikita55612/goTradingBot/internal/trading/predict"
	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
	"github.com/nikita55612/goTradingBot/internal/utils/seqs"
)

type TrendStrategy struct {
	symbol   string
	interval cdl.Interval

	qtyPrecision      int
	minOrderAmt       float64
	tickSize          float64
	tickSizePrecision int

	ctx              context.Context
	subData          *trading.SubData
	orderRequestChan chan<- *trading.OrderRequest

	candleStreamChan chan *cdl.CandleStreamData
	candleStream     chan<- struct{}

	orderUpdateChan    chan *trading.OrderUpdate
	confirmHandlerChan chan *cdl.Candle
	backgroundChan     chan *cdl.Candle

	currencyVolume float64
	longRatio      float64

	orderLog        *seqs.OrderedMap[string, *trading.Order]
	orderExecQtyLog map[string]float64
	qtyPosition     atomic.Pointer[float64]

	limitOrderOffset float64

	lastPrice       atomic.Pointer[float64]
	limitCeilPrice  atomic.Pointer[float64]
	limitFloorPrice atomic.Pointer[float64]

	trendPredictor *predict.TrendPredictor

	isWorking atomic.Bool
}

func NewTrendStrategy(
	symbol string,
	interval cdl.Interval,
	currencyVolume float64,
	longRatio float64,
	limitOrderOffset float64,
) *TrendStrategy {
	s := &TrendStrategy{
		symbol:           symbol,
		interval:         interval,
		currencyVolume:   currencyVolume,
		longRatio:        longRatio,
		limitOrderOffset: limitOrderOffset,
		trendPredictor:   predict.NewTrendPredictor(interval),
		orderLog:         seqs.NewOrderedMap[string, *trading.Order](1000),
	}
	return s
}

func (s *TrendStrategy) Init(ctx context.Context, subData *trading.SubData, req chan<- *trading.OrderRequest) {
	s.ctx = ctx
	s.subData = subData
	s.orderRequestChan = req

	go func() {
		<-s.ctx.Done()
		s.Stop()
	}()
}

func (s *TrendStrategy) getConfirmCandles(limit int) ([]cdl.Candle, error) {
	return s.subData.GetCandles(s.symbol, s.interval, limit)
}

func (s *TrendStrategy) Work() error {
	if s.isWorking.Load() {
		return nil
	}
	instrumentInfo, err := s.subData.GetInstrumentInfo(s.symbol)
	if err != nil {
		return err
	}
	s.qtyPrecision = instrumentInfo.QtyPrecision
	s.minOrderAmt = instrumentInfo.MinOrderAmt
	s.tickSize = instrumentInfo.TickSize
	s.tickSizePrecision = numeric.DecimalPlaces(s.tickSize)

	lastCandle, err := s.getConfirmCandles(1)
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
	s.orderUpdateChan = make(chan *trading.OrderUpdate)
	s.confirmHandlerChan = make(chan *cdl.Candle)
	s.backgroundChan = make(chan *cdl.Candle)

	s.orderExecQtyLog = make(map[string]float64)
	qtyPosition := 0.
	s.qtyPosition.Store(&qtyPosition)

	candles, err := s.getConfirmCandles(predict.TpIBS)
	if err != nil {
		close(s.candleStreamChan)
		return err
	}
	if err := s.trendPredictor.Init(candles); err != nil {
		close(s.candleStreamChan)
		return err
	}

	go s.orderUpdate()
	go s.background()
	go s.confirmHandler()
	go s.observe()

	s.isWorking.Store(true)

	return nil
}

func (s *TrendStrategy) Stop() bool {
	if !s.isWorking.Load() {
		return false
	}
	qtyPosition := *s.qtyPosition.Load()
	if qtyPosition != 0 {
		s.orderRequestChan <- trading.NewOrderRequest(
			trading.NewOrder(s.symbol, -qtyPosition, nil),
		)
	}
	close(s.candleStream)
	close(s.confirmHandlerChan)
	close(s.backgroundChan)
	close(s.orderUpdateChan)
	s.isWorking.Store(false)
	return true
}

func (s *TrendStrategy) orderUpdate() {
	for update := range s.orderUpdateChan {
		execQty := update.Order.ExecQty
		if execQty != 0 {
			if eq, ok := s.orderExecQtyLog[update.LinkId]; ok {
				execQty -= eq
			}
			s.orderExecQtyLog[update.LinkId] = update.Order.ExecQty
			if execQty != 0 {
				qtyPosition := *s.qtyPosition.Load()
				qtyPosition += execQty
				qtyPosition = numeric.TruncateFloat(qtyPosition, s.qtyPrecision)
				s.qtyPosition.Store(&qtyPosition)
			}
		}
	}
}

func (s *TrendStrategy) observe() {
	for data := range s.candleStreamChan {
		s.backgroundChan <- &data.Candle
		if data.Confirm {
			s.confirmHandlerChan <- &data.Candle
		}
	}
}

func (s *TrendStrategy) background() {
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

func (s *TrendStrategy) confirmHandler() {
	for candle := range s.confirmHandlerChan {
		_ = candle

		candles, err := s.getConfirmCandles(90)
		if err != nil {
			log.Printf("get confirm candles error: %s", err)
			continue
		}
		p, err := s.trendPredictor.GetNextPrediction(candles)
		if err != nil {
			log.Printf("get next prediction error: %s", err)
			continue
		}
		fmt.Printf("%s Prediction: %v\n", s.symbol, p)

		if p[1] == 0 {
			continue
		}

		qtyVolume := s.currencyVolume / *s.lastPrice.Load()

		var directedQty float64
		if p[1] > .5 {
			if p[0] > .5 {
				directedQty = qtyVolume * s.longRatio
			} else {
				directedQty = -qtyVolume * (1 - s.longRatio)
			}
		}

		qty := -*s.qtyPosition.Load() + directedQty
		qty = numeric.RoundFloat(qty, s.qtyPrecision)
		if math.Abs(qty**s.lastPrice.Load()) < s.minOrderAmt {
			log.Printf("qty less than minimum limit: %s", err)
			continue
		}

		var price float64
		if qty > 0 {
			price = *s.limitCeilPrice.Load()
		} else {
			price = *s.limitFloorPrice.Load()
		}

		order := trading.NewOrder(s.symbol, qty, &price)
		linkId := uuid.NewString()
		s.orderLog.Set(linkId, order)
		s.orderRequestChan <- trading.NewOrderRequest(
			order,
			trading.WithLinkId(linkId),
		)
	}
}
