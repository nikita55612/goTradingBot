package broker

import (
	"context"

	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
)

type Broker interface {
	GetInstrumentInfo(symbol string) ([]byte, error)
	GetCandles(symbol string, interval cdl.Interval, limit int) ([]cdl.Candle, error)
	CandleStream(ctx context.Context, symbol string, interval cdl.Interval) (<-chan *cdl.CandleStreamData, error)
	PlaceOrder(symbol string, qty float64, price *float64) (string, error)
	CancelOrder(orderId string) (string, error)
	GetOrder(orderId string) ([]byte, error)
}
