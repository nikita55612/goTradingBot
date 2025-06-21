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

	availableBalance float64
	longRatio        float64

	// _PnL            float64
	orderLog         *seqs.OrderedMap[string, *trading.Order]
	qtyPosition      atomic.Pointer[float64]
	avgPositionPrice atomic.Pointer[float64]

	trendPredictor  *predict.TrendPredictor
	trendZoneFilter float64

	martngaleSteps []float64
	longLosses     atomic.Pointer[int]
	shortLosses    atomic.Pointer[int]

	limitOrderOffset float64

	lastPrice       atomic.Pointer[float64]
	limitCeilPrice  atomic.Pointer[float64]
	limitFloorPrice atomic.Pointer[float64]

	lastOrderRequestTime int64
	isWorking            atomic.Bool
}

func NewTrendStrategy(cfg *trading.StrategyConfig) (*TrendStrategy, error) {
	if cfg.Symbol == "" {
		return nil, fmt.Errorf("symbol not specified in configuration parameters")
	}

	mrPrefix := []float64{1}
	var martngaleRatios []float64
	if cfg.MartngaleRatios != nil {
		martngaleRatios = append(mrPrefix, cfg.MartngaleRatios...)
	} else {
		martngaleRatios = mrPrefix
	}

	balance := cfg.AvailableBalance
	martngaleSteps := make([]float64, len(martngaleRatios))
	for i := len(martngaleRatios) - 1; i >= 0; i-- {
		martngaleSteps[i] = balance
		balance /= martngaleRatios[i]
	}

	interval, err := cdl.ParseInterval(cfg.Interval)
	if err != nil {
		return nil, err
	}

	switch interval {
	case cdl.M5:
	case cdl.M15:
	default:
		return nil, fmt.Errorf("strategy does not support interval: %s", interval.AsString())
	}

	var longRatio float64
	if cfg.LongRatio == nil {
		longRatio = .5
	} else {
		longRatio = *cfg.LongRatio
		if longRatio > 1 {
			longRatio = 1
		}
		if longRatio < 0 {
			longRatio = 0
		}
	}

	var trendZoneFilter float64
	if cfg.TrendZoneFilter == nil {
		trendZoneFilter = .5
	} else {
		trendZoneFilter = *cfg.TrendZoneFilter
		if trendZoneFilter > .7 {
			trendZoneFilter = .7
		}
		if trendZoneFilter < 0 {
			trendZoneFilter = 0
		}
	}

	var limitOrderOffset float64
	if cfg.LimitOrderOffset == nil {
		limitOrderOffset = .01
	} else {
		limitOrderOffset = *cfg.LimitOrderOffset
		if limitOrderOffset > 0.1 {
			limitOrderOffset = .1
		}
		if limitOrderOffset < 0 {
			limitOrderOffset = .001
		}
	}

	s := &TrendStrategy{
		symbol:           cfg.Symbol,
		interval:         interval,
		availableBalance: cfg.AvailableBalance,
		longRatio:        longRatio,
		martngaleSteps:   martngaleSteps,
		trendZoneFilter:  trendZoneFilter,
		limitOrderOffset: limitOrderOffset,
	}

	return s, nil
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

func (s *TrendStrategy) readConfirmCandles(limit int) ([]cdl.Candle, error) {
	return s.subData.ReadConfirmCandles(s.symbol, s.interval, limit)
}

func (s *TrendStrategy) Launch() (err error) {
	if !s.isWorking.CompareAndSwap(false, true) {
		return err
	}

	defer func() {
		if err != nil {
			s.isWorking.Store(false)
		}
	}()

	s.trendPredictor = predict.NewTrendPredictor(s.interval)
	instrumentInfo, err := s.subData.GetInstrumentInfo(s.symbol)
	if err != nil {
		return err
	}
	s.qtyPrecision = instrumentInfo.QtyPrecision
	s.minOrderAmt = instrumentInfo.MinOrderAmt
	s.tickSize = instrumentInfo.TickSize
	s.tickSizePrecision = numeric.DecimalPlaces(s.tickSize)

	if s.martngaleSteps[0] < s.minOrderAmt {
		if len(s.martngaleSteps) > 1 {
			err = fmt.Errorf(
				"martingale step is less than the minimum order amt: %f < %f",
				s.martngaleSteps[0],
				s.minOrderAmt,
			)
			return err
		} else {
			err = fmt.Errorf(
				"available balance is less than the minimum order amt: %f < %f",
				s.martngaleSteps[0],
				s.minOrderAmt,
			)
			return err
		}
	}

	longAmt := s.martngaleSteps[0] * s.longRatio
	if longAmt > 0 && longAmt < s.minOrderAmt {
		err = fmt.Errorf(
			"long amt is less than the minimum order amt: %f < %f",
			longAmt,
			s.minOrderAmt,
		)
		return err
	}
	shortAmt := s.martngaleSteps[0] * (1 - s.longRatio)
	if longAmt > 0 && longAmt < s.minOrderAmt {
		err = fmt.Errorf(
			"short amt is less than the minimum order amt: %f < %f",
			shortAmt,
			s.minOrderAmt,
		)
		return err
	}

	longLosses := 0
	s.longLosses.Store(&longLosses)
	shortLosses := 0
	s.shortLosses.Store(&shortLosses)

	lastCandle, err := s.readConfirmCandles(1)
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

	s.orderLog = seqs.NewOrderedMap[string, *trading.Order](100)
	qtyPosition := 0.
	s.qtyPosition.Store(&qtyPosition)
	avgPricePosition := 0.
	s.avgPositionPrice.Store(&avgPricePosition)

	candles, err := s.readConfirmCandles(predict.TpIBS)
	if err != nil {
		close(s.candleStream)
		return err
	}
	if err := s.trendPredictor.Init(candles); err != nil {
		close(s.candleStream)
		return err
	}

	go s.orderUpdate()
	go s.background()
	go s.confirmHandler()
	go s.observe()

	return nil
}

func (s *TrendStrategy) Stop() bool {
	if !s.isWorking.CompareAndSwap(true, false) {
		return false
	}

	close(s.confirmHandlerChan)

	timeNow := time.Now().UnixMilli()
	if timeNow-s.lastOrderRequestTime < 500 {
		time.Sleep(300 * time.Millisecond)
	}

	qtyPosition := *s.qtyPosition.Load()
	if qtyPosition != 0 {
		s.orderRequestChan <- trading.NewOrderRequest(
			trading.NewOrder(s.symbol, -qtyPosition, nil),
		)
	}

	close(s.candleStream)
	close(s.backgroundChan)
	close(s.orderUpdateChan)

	return true
}

func (s *TrendStrategy) orderUpdate() {
	for update := range s.orderUpdateChan {
		if update.Order.ID == "" {
			continue
		}

		execQty := update.Order.ExecQty
		if execQty == 0 {
			continue
		}

		if o, ok := s.orderLog.Get(update.LinkId); ok {
			execQty -= o.ExecQty
		}

		prevQtyPosition := *s.qtyPosition.Load()
		qtyPosition := prevQtyPosition + execQty
		qtyPosition = numeric.TruncateFloat(qtyPosition, s.qtyPrecision)
		s.qtyPosition.Store(&qtyPosition)

		s.orderLog.Set(update.LinkId, update.Order)

		if s.orderLog.Len() == 1 {
			s.avgPositionPrice.Store(&update.Order.AvgPrice)
			continue
		}
		if (prevQtyPosition > 0) == (qtyPosition > 0) && qtyPosition != 0 {
			avgPrice := *s.avgPositionPrice.Load()
			newAvgPrice := (avgPrice + update.Order.AvgPrice) / 2
			newAvgPrice = numeric.TruncateFloat(newAvgPrice, s.tickSizePrecision)
			s.avgPositionPrice.Store(&newAvgPrice)
			continue
		}

		prevAvgPrice := *s.avgPositionPrice.Load()
		s.avgPositionPrice.Store(&update.Order.AvgPrice)

		if prevQtyPosition > 0 {
			if prevAvgPrice > update.Order.AvgPrice {
				v := *s.longLosses.Load() + 1
				s.longLosses.Store(&v)
			} else {
				v := 0
				s.longLosses.Store(&v)
			}
		}

		if prevQtyPosition < 0 {
			if prevAvgPrice < update.Order.AvgPrice {
				v := *s.shortLosses.Load() + 1
				s.shortLosses.Store(&v)
			} else {
				v := 0
				s.shortLosses.Store(&v)
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

		candles, err := s.readConfirmCandles(90)
		if err != nil {
			log.Printf("get confirm candles error: %s", err)
			continue
		}

		p, err := s.trendPredictor.GetNextPrediction(candles)
		if err != nil {
			log.Printf("get next prediction error: %s", err)
			continue
		}

		if p[1] == 0 {
			continue
		}

		var directedQty float64
		if p[1] > s.trendZoneFilter {
			if p[0] > .5 {
				longLosses := min(len(s.martngaleSteps)-1, *s.longLosses.Load())
				qtyVolume := s.martngaleSteps[longLosses] / *s.lastPrice.Load()
				directedQty = qtyVolume * s.longRatio
			} else {
				shortLosses := min(len(s.martngaleSteps)-1, *s.shortLosses.Load())
				qtyVolume := s.martngaleSteps[shortLosses] / *s.lastPrice.Load()
				directedQty = -qtyVolume * (1 - s.longRatio)
			}
		}

		qtyPosition := *s.qtyPosition.Load()

		if qtyPosition == 0 && directedQty == 0 {
			continue
		}

		qty := -qtyPosition + directedQty
		qty = numeric.RoundFloat(qty, s.qtyPrecision)
		if absQty := math.Abs(qty * *s.lastPrice.Load()); absQty < s.minOrderAmt {
			log.Printf("qty less than minimum limit: %f < %f", absQty, s.minOrderAmt)
			continue
		}

		linkId := uuid.NewString()

		var price float64
		if qty > 0 {
			price = *s.limitCeilPrice.Load()
		} else {
			price = *s.limitFloorPrice.Load()
		}

		order := trading.NewOrder(s.symbol, qty, &price)
		if s.isWorking.Load() {
			s.lastOrderRequestTime = time.Now().UnixMilli()
			s.orderRequestChan <- trading.NewOrderRequest(
				order,
				trading.WithLinkId(linkId),
				trading.WithReply(s.orderUpdateChan),
			)
		}
	}
}
