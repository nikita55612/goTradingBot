package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nikita55612/goTradingBot/internal/broker/bybit/models"
	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/pkg/ws"
	"github.com/nikita55612/httpx"
)

// GetInstrumentInfo возвращает информацию о торговом инструменте по его символу.
// https://bybit-exchange.github.io/docs/v5/market/instrument
func (c *Client) GetInstrumentInfo(symbol string) (*models.InstrumentInfo, error) {
	query := make(url.Values)
	query.Set("category", c.category)
	query.Set("symbol", symbol)
	queryString := query.Encode()
	path := fmt.Sprintf(
		"%s%s?%s",
		c.baseURL,
		"/v5/market/instruments-info",
		queryString,
	)
	req := httpx.Get(path)
	var instrumentInfoResult models.InstrumentInfoResult
	if err := c.callAPI(req, queryString, &instrumentInfoResult); err != nil {
		return nil, err.(*Error).SetEndpoint("GetInstrumentInfo")
	}
	var instrumentInfo models.InstrumentInfo
	if len(instrumentInfoResult.List) > 0 {
		instrumentInfo = instrumentInfoResult.List[0]
	}

	return &instrumentInfo, nil
}

// GetCandles возвращает исторические свечи с ограничением по количеству. Последняя свеча не подтверждена.
// https://bybit-exchange.github.io/docs/v5/market/kline
func (c *Client) GetCandles(symbol string, interval cdl.Interval, limit int) ([]cdl.Candle, error) {
	query := make(url.Values)
	query.Set("category", c.category)
	query.Set("symbol", symbol)
	query.Set("interval", AsLocalInterval(interval))
	query.Set("limit", strconv.Itoa(min(limit, 1000)))
	res, err := c.getCandle(query)
	if err != nil {
		return nil, err
	}

	candles, extractErr := extractCandleFromResult(res)
	if extractErr != nil {
		return candles, extractErr
	}

	counter := limit - 1000
	for counter > 0 {
		nextLimit := min(1000, counter)
		end := candles[len(candles)-1].Time - 1
		query.Set("limit", strconv.Itoa(nextLimit))
		query.Set("end", strconv.FormatInt(end, 10))
		res, err := c.getCandle(query)
		if err != nil {
			return candles, err
		}
		newCandles, extractErr := extractCandleFromResult(res)
		if extractErr != nil {
			return candles, extractErr
		}
		candles = append(candles, newCandles...)
		counter -= nextLimit
		if counter == 0 {
			break
		}
	}
	slices.Reverse(candles)

	return candles, nil
}

// CandleStream устанавливает WebSocket соединение для потокового получения свечей.
// https://bybit-exchange.github.io/docs/v5/websocket/public/kline
func (c *Client) CandleStream(ctx context.Context, symbol string, interval cdl.Interval) (<-chan *cdl.CandleStreamData, error) {
	arg := fmt.Sprintf("kline.%s.%s", AsLocalInterval(interval), symbol)
	subMessage := map[string]any{
		"req_id": uuid.NewString(),
		"op":     "subscribe",
		"args":   []string{arg},
	}
	handshakeMessage, _ := json.Marshal(subMessage)
	outChan, err := ws.Connect(
		fmt.Sprintf("%s/%s", PUBLICWS, c.category),
		ctx,
		ws.WithHandshake(handshakeMessage),
	)
	if err != nil {
		err = fmt.Errorf("failed to create websocket connection: %w", err)
		return nil, NewError(InternalErrorT, err).SetEndpoint("CandleStream")
	}

	stream := make(chan *cdl.CandleStreamData)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(stream)
				return
			case data := <-outChan:
				var candleStreamRawData models.CandleStreamRawData
				if err := json.Unmarshal(data, &candleStreamRawData); err != nil {
					continue
				}
				candleStreamData, err := candleStreamFromRawData(&candleStreamRawData)
				if err != nil {
					continue
				}
				select {
				case stream <- candleStreamData:
				case <-time.After(time.Second):
					if candleStreamData.Confirm {
						stream <- candleStreamData
					}
				}

			}
		}
	}()

	return stream, nil
}

// getCandles выполняет запрос исторических данных свечей
func (c *Client) getCandle(query url.Values) (*models.CandleResult, *Error) {
	queryString := query.Encode()
	path := fmt.Sprintf(
		"%s%s?%s",
		c.baseURL,
		"/v5/market/kline",
		queryString,
	)
	req := httpx.Get(path)
	var candleResult models.CandleResult
	if err := c.callAPI(req, queryString, &candleResult); err != nil {
		return &candleResult, err.(*Error).SetEndpoint("getCandle")
	}

	return &candleResult, nil
}
