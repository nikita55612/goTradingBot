package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
)

type DataProvider interface {
	cdl.CandleProvider
	GetInstrumentInfo(symbol string) ([]byte, error)
}

type InstrumentInfo struct {
	QtyPrecision int     `json:"qtyPrecision"`
	MinOrderAmt  float64 `json:"minOrderAmt"`
	TickSize     float64 `json:"tickSize"`
}

type SubData struct {
	dataProvider DataProvider
	candleSyncs  map[string]*cdl.CandleSync
	bufferSize   int
	ctx          context.Context
	mu           sync.Mutex
}

func NewSubData(ctx context.Context, dataProvider DataProvider, bufferSize int) *SubData {
	return &SubData{
		dataProvider: dataProvider,
		candleSyncs:  make(map[string]*cdl.CandleSync),
		bufferSize:   bufferSize,
		ctx:          ctx,
	}
}

func (s *SubData) getCandleSync(symbol string, interval cdl.Interval) (*cdl.CandleSync, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s-%d", symbol, interval)
	if candleSync, ok := s.candleSyncs[key]; ok {
		return candleSync, nil
	}
	newCandleSync := cdl.NewCandleSync(s.ctx, symbol, interval, s.bufferSize, s.dataProvider)
	if err := newCandleSync.StartSync(); err != nil {
		return nil, err
	}
	s.candleSyncs[key] = newCandleSync
	return newCandleSync, nil
}

func (s *SubData) SubscribeChan(symbol string, interval cdl.Interval, ch chan<- *cdl.CandleStreamData) (chan<- struct{}, error) {
	candleSync, err := s.getCandleSync(symbol, interval)
	if err != nil {
		return nil, err
	}
	return candleSync.Subscribe(ch), nil
}

func (s *SubData) ReadConfirmCandles(symbol string, interval cdl.Interval, limit int) ([]cdl.Candle, error) {
	candleSync, err := s.getCandleSync(symbol, interval)
	if err != nil {
		return nil, err
	}
	return candleSync.ReadConfirmCandles(limit), nil
}

func (s *SubData) GetCandles(symbol string, interval cdl.Interval, limit int) ([]cdl.Candle, error) {
	return s.dataProvider.GetCandles(symbol, interval, limit)
}

func (s *SubData) GetInstrumentInfo(symbol string) (*InstrumentInfo, error) {
	b, err := s.dataProvider.GetInstrumentInfo(symbol)
	if err != nil {
		return nil, err
	}
	var instrumentInfo InstrumentInfo
	if err = json.Unmarshal(b, &instrumentInfo); err != nil {
		return nil, err
	}
	return &instrumentInfo, nil
}

func (s *SubData) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k := range s.candleSyncs {
		delete(s.candleSyncs, k)
	}
}
