package trading

import (
	"sync"
	"time"
)

type Order struct {
	sync.Mutex `json:"-"`
	ID         string   `json:"id"`        // ID ордера
	Symbol     string   `json:"symbol"`    // Торговая пара
	Qty        float64  `json:"qty"`       // Исходное количество
	Price      *float64 `json:"price"`     // Цена для лимитного ордера
	AvgPrice   float64  `json:"avgPrice"`  // Средняя цена исполнения
	ExecQty    float64  `json:"execQty"`   // Исполненное количество
	ExecValue  float64  `json:"execValue"` // Стоимость исполненного объема
	Fee        float64  `json:"fee"`       // Сумма комиссии
	CreatedAt  int64    `json:"createdAt"` // Время создания (мс)
	UpdatedAt  int64    `json:"updatedAt"` // Время обновления (мс)
	IsClosed   bool     `json:"isClosed"`  // Флаг завершенности
}

func NewOrder(symbol string, qty float64, price *float64) *Order {
	return &Order{
		Symbol:    symbol,
		Qty:       qty,
		Price:     price,
		CreatedAt: time.Now().UnixMilli(),
	}
}

func (o *Order) Replace(newOrder *Order) {
	o.AvgPrice = newOrder.AvgPrice
	o.Qty = newOrder.Qty
	o.Price = newOrder.Price
	o.ExecQty = newOrder.ExecQty
	o.ExecValue = newOrder.ExecValue
	o.Fee = newOrder.Fee
	o.CreatedAt = newOrder.CreatedAt
	o.UpdatedAt = newOrder.UpdatedAt
	o.ID = newOrder.ID
	o.Symbol = newOrder.Symbol
	o.IsClosed = newOrder.IsClosed
}

func (o *Order) Clone() *Order {
	return &Order{
		AvgPrice:  o.AvgPrice,
		Qty:       o.Qty,
		Price:     o.Price,
		ExecQty:   o.ExecQty,
		ExecValue: o.ExecValue,
		Fee:       o.Fee,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
		ID:        o.ID,
		Symbol:    o.Symbol,
		IsClosed:  o.IsClosed,
	}
}

type OrderUpdate struct {
	LinkId string `json:"linkId"`
	Order  *Order `json:"order"`
}

type OrderRequest struct {
	LinkId       string              `json:"linkId"`
	Tag          string              `json:"tag"`
	Order        *Order              `json:"order"`
	Delay        time.Duration       `json:"-"`
	PlaceTimeout time.Duration       `json:"-"`
	CloseTimeout time.Duration       `json:"-"`
	Reply        chan<- *OrderUpdate `json:"-"`
}

func NewOrderRequest(order *Order, opts ...OrderRequestOption) *OrderRequest {
	r := &OrderRequest{
		Order:        order,
		PlaceTimeout: 2 * time.Second,
		CloseTimeout: 1 * time.Minute,
	}
	for _, option := range opts {
		option(r)
	}
	return r

}

type OrderRequestOption func(*OrderRequest)

func WithLinkId(linkId string) OrderRequestOption {
	return func(r *OrderRequest) {
		r.LinkId = linkId
	}
}

func WithTag(tag string) OrderRequestOption {
	return func(r *OrderRequest) {
		r.Tag = tag
	}
}

func WithDelay(d time.Duration) OrderRequestOption {
	return func(r *OrderRequest) {
		r.Delay = d
	}
}

func WithPlaceTimeout(d time.Duration) OrderRequestOption {
	return func(r *OrderRequest) {
		r.PlaceTimeout = d
	}
}

func WithCloseTimeout(d time.Duration) OrderRequestOption {
	return func(r *OrderRequest) {
		r.CloseTimeout = d
	}
}

func WithReply(reply chan<- *OrderUpdate) OrderRequestOption {
	return func(r *OrderRequest) {
		r.Reply = reply
	}
}

func (r *OrderRequest) Clone() *OrderRequest {
	var clonedOrder *Order
	if r.Order != nil {
		clonedOrder = r.Order.Clone()
	}
	return &OrderRequest{
		LinkId:       r.LinkId,
		Tag:          r.Tag,
		Order:        clonedOrder,
		Delay:        r.Delay,
		PlaceTimeout: r.PlaceTimeout,
		CloseTimeout: r.CloseTimeout,
		Reply:        r.Reply,
	}
}
