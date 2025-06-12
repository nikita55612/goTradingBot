package bybit

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit/models"
	"github.com/nikita55612/httpx"
)

// PlaceOrder создает рыночный или лимитный ордер.
// https://bybit-exchange.github.io/docs/v5/order/create-order
func (c *Client) PlaceOrder(symbol string, qty float64, price *float64) (string, error) {
	params := map[string]any{
		"category":   c.category,
		"symbol":     symbol,
		"side":       "Buy",
		"orderType":  "Market",
		"isLeverage": 1,
	}
	if qty < 0 {
		params["side"] = "Sell"
	}
	params["qty"] = strconv.FormatFloat(math.Abs(qty), 'f', -1, 64)
	if price != nil {
		params["price"] = strconv.FormatFloat(*price, 'f', -1, 64)
		params["orderType"] = "Limit"
	}
	jsonData, _ := json.Marshal(params)
	fullURL := fmt.Sprintf("%s%s", c.baseURL, "/v5/order/create")
	req := httpx.Post(fullURL).WithData(jsonData)
	var placeOrderResult models.PlaceOrderResult
	if err := c.callAPI(req, string(jsonData), &placeOrderResult); err != nil {
		return "", err.(*Error).SetEndpoint("PlaceOrder")
	}
	return placeOrderResult.OrderId, nil
}

// CancelOrder отменяет активный ордер.
// https://bybit-exchange.github.io/docs/v5/order/cancel-order
func (c *Client) CancelOrder(orderId string) (string, error) {
	query := make(url.Values)
	query.Set("category", c.category)
	query.Set("orderId", orderId)
	queryString := query.Encode()
	fullURL := fmt.Sprintf("%s%s?%s", c.baseURL, "/v5/order/cancel", queryString)
	req := httpx.Post(fullURL)
	var cancelOrderResult models.CancelOrderResult
	if err := c.callAPI(req, queryString, &cancelOrderResult); err != nil {
		return "", err.(*Error).SetEndpoint("CancelOrder")
	}
	return cancelOrderResult.OrderId, nil
}

// GetOrderHistoryDetail возвращает детали ордера по его ID.
// https://bybit-exchange.github.io/docs/v5/order/order-list
func (c *Client) GetOrderHistoryDetail(orderId string) (*models.OrderHistoryDetail, *Error) {
	query := make(url.Values)
	query.Set("category", c.category)
	query.Set("orderId", orderId)
	queryString := query.Encode()
	fullURL := fmt.Sprintf("%s%s?%s", c.baseURL, "/v5/order/history", queryString)
	req := httpx.Get(fullURL)
	var orderHistoryResult models.OrderHistoryResult
	if err := c.callAPI(req, queryString, &orderHistoryResult); err != nil {
		return nil, err.(*Error).SetEndpoint("GetOrderHistoryDetail")
	}
	if len(orderHistoryResult.List) == 0 {
		err := fmt.Errorf("order with id %s not found", orderId)
		return nil, NewError(InternalErrorT, err).SetEndpoint("GetOrderHistoryDetail")
	}
	return &orderHistoryResult.List[0], nil
}
