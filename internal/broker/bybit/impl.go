package bybit

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/nikita55612/goTradingBot/internal/broker"
	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
)

func (c *Client) BrokerImpl() broker.Broker {
	return &BrokerImpl{cli: c}
}

type BrokerImpl struct {
	cli *Client
}

func (b *BrokerImpl) GetInstrumentInfo(symbol string) ([]byte, error) {
	info, err := b.cli.GetInstrumentInfo(symbol)
	if err != nil {
		return nil, err
	}

	var minOrderAmt float64
	var qtyPrecision int
	if b.cli.category == "spot" {
		v, parseErr := strconv.ParseFloat(info.LotSizeFilter.MaxOrderAmt, 64)
		if parseErr != nil {
			return nil, parseErr
		}
		minOrderAmt = v
		v, parseErr = strconv.ParseFloat(info.LotSizeFilter.BasePrecision, 64)
		if parseErr != nil {
			return nil, parseErr
		}
		qtyPrecision = numeric.DecimalPlaces(v)
	} else {
		v, parseErr := strconv.ParseFloat(info.LotSizeFilter.MinNotionalValue, 64)
		if parseErr != nil {
			return nil, parseErr
		}
		minOrderAmt = v
		v, parseErr = strconv.ParseFloat(info.LotSizeFilter.QtyStep, 64)
		if parseErr != nil {
			return nil, parseErr
		}
		qtyPrecision = numeric.DecimalPlaces(v)
	}

	tickSize, parseErr := strconv.ParseFloat(info.PriceFilter.TickSize, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	infoData := map[string]any{
		"qtyPrecision": qtyPrecision,
		"minOrderAmt":  minOrderAmt,
		"tickSize":     tickSize,
	}

	return json.Marshal(infoData)
}

func (b *BrokerImpl) GetCandles(symbol string, interval cdl.Interval, limit int) ([]cdl.Candle, error) {
	return b.cli.GetCandles(symbol, interval, limit)
}

func (b *BrokerImpl) CandleStream(ctx context.Context, symbol string, interval cdl.Interval) (<-chan *cdl.CandleStreamData, error) {
	return b.cli.CandleStream(ctx, symbol, interval)
}

func (b *BrokerImpl) PlaceOrder(symbol string, qty float64, price *float64) (string, error) {
	return b.cli.PlaceOrder(symbol, qty, price)
}

func (b *BrokerImpl) CancelOrder(orderId string) (string, error) {
	return b.cli.CancelOrder(orderId)
}

func (b *BrokerImpl) GetOrder(orderId string) ([]byte, error) {
	detail, err := b.cli.GetOrderHistoryDetail(orderId)
	if err != nil {
		return nil, err
	}
	createdAt, parseErr := strconv.ParseInt(detail.CreatedTime, 10, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	updatedAt, parseErr := strconv.ParseInt(detail.UpdatedTime, 10, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	qty, parseErr := strconv.ParseFloat(detail.Qty, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	price, parseErr := strconv.ParseFloat(detail.Price, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	avgPrice, parseErr := strconv.ParseFloat(detail.AvgPrice, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	execQty, parseErr := strconv.ParseFloat(detail.CumExecQty, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	execValue, parseErr := strconv.ParseFloat(detail.CumExecValue, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	if detail.Side == "Sell" {
		qty = -qty
		execQty = -execQty
		execValue = -execValue
	}
	fee, parseErr := strconv.ParseFloat(detail.CumExecFee, 64)
	if parseErr != nil {
		return nil, parseErr
	}
	isClosed := true
	switch detail.OrderStatus {
	case "New", "PartiallyFilled", "Untriggered":
		isClosed = false
	}
	orderData := map[string]any{
		"id":        detail.OrderId,
		"symbol":    detail.Symbol,
		"qty":       qty,
		"price":     price,
		"avgPrice":  avgPrice,
		"execQty":   execQty,
		"execValue": execValue,
		"fee":       fee,
		"isClosed":  isClosed,
		"createdAt": createdAt,
		"updatedAt": updatedAt,
	}

	return json.Marshal(orderData)
}
