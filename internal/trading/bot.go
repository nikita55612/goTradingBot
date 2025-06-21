package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nikita55612/goTradingBot/internal/broker"
	"github.com/nikita55612/goTradingBot/internal/utils/slogx"
)

type Strategy interface {
	Init(ctx context.Context, subData *SubData, req chan<- *OrderRequest)
	Launch() error
	Stop() bool
}

type TradingBot struct {
	ctx              context.Context
	broker           broker.Broker
	subData          *SubData
	logger           *slogx.AsyncSlog
	orderRequestChan chan *OrderRequest
	strategies       map[string]Strategy
	strategiesMu     sync.Mutex
}

func NewTradingBot(ctx context.Context, broker broker.Broker, logger *slog.Logger) *TradingBot {
	var asyncSlog *slogx.AsyncSlog
	if logger != nil {
		asyncSlog = slogx.NewAsyncSlog(context.Background(), logger)
	}

	b := &TradingBot{
		ctx:              ctx,
		broker:           broker,
		subData:          NewSubData(ctx, broker, 1000),
		logger:           asyncSlog,
		orderRequestChan: make(chan *OrderRequest),
		strategies:       make(map[string]Strategy),
	}

	go func() {
		<-ctx.Done()
		b.Stop()
	}()

	go b.orderRequestHandler()

	b.log(slog.LevelInfo, "trading bot start polling")

	return b
}

func (b *TradingBot) log(level slog.Level, msg string, args ...any) {
	if b.logger != nil {
		b.logger.Log(level, msg, args...)
	}
}

func (b *TradingBot) Stop() {
	b.strategiesMu.Lock()
	defer b.strategiesMu.Unlock()

	for _, s := range b.strategies {
		s.Stop()
	}

	b.log(slog.LevelInfo, "trading bot stopped")

}

func (b *TradingBot) Resume() error {
	b.strategiesMu.Lock()
	defer b.strategiesMu.Unlock()

	var err error
	for _, s := range b.strategies {
		if e := s.Launch(); e != nil {
			err = e
		}
	}
	if err == nil {
		b.log(slog.LevelInfo, "trading bot start polling")
	}

	return err
}

func (b *TradingBot) StopStrategy(id string) bool {
	b.strategiesMu.Lock()
	defer b.strategiesMu.Unlock()

	if s, ok := b.strategies[id]; ok {
		return s.Stop()
	}
	return false
}

func (b *TradingBot) LaunchStrategy(id string) error {
	b.strategiesMu.Lock()
	defer b.strategiesMu.Unlock()

	if s, ok := b.strategies[id]; ok {
		return s.Launch()
	}
	return nil
}

func (b *TradingBot) AddStrategy(s Strategy) (string, error) {
	b.strategiesMu.Lock()
	defer b.strategiesMu.Unlock()

	s.Init(context.Background(), b.subData, b.orderRequestChan)
	strategyID := uuid.NewString()
	b.strategies[strategyID] = s

	return strategyID, nil
}

func (b *TradingBot) replyOrder(req *OrderRequest) {
	if req.Reply == nil {
		return
	}
	select {
	case req.Reply <- &OrderUpdate{
		LinkId: req.LinkId,
		Tag:    req.Tag,
		Order:  req.Order.Clone(),
	}:
	case <-time.After(time.Second):
	}
}

func (b *TradingBot) orderRequestHandler() {
	for req := range b.orderRequestChan {
		go func() {
			if !b.placeOrderWithRetry(req) {
				return
			}
			b.replyOrder(req)

			var err error
			if !b.waitForOrderClosed(req) {
				err = fmt.Errorf("waiting time for order closing has expired")
				if !b.cancelOrderWithRetry(req) {
					err = fmt.Errorf("failed to cancel unclosed order")
				}
			}
			b.replyOrder(req)

			logLevel := slog.LevelInfo
			if err != nil {
				logLevel = slog.LevelError
			}
			b.log(
				logLevel,
				"order processing completed",
				"error", err,
				"orderRequest", req.Clone(),
			)
		}()
	}
}

func (b *TradingBot) placeOrderWithRetry(req *OrderRequest) bool {
	if req.Delay > 0 {
		time.Sleep(req.Delay)
	}
	timeout := time.After(req.PlaceTimeout)
	for {
		req.Order.Lock()
		orderId, err := b.broker.PlaceOrder(
			req.Order.Symbol,
			req.Order.Qty,
			req.Order.Price,
		)
		if err == nil {
			req.Order.ID = orderId
			req.Order.Unlock()
			return true
		}
		req.Order.Unlock()

		select {
		case <-time.After(100 * time.Millisecond):
		case <-timeout:
			b.log(
				slog.LevelError,
				"order registration deadline has expired",
				"error", err,
				"orderRequest", req.Clone(),
			)
			return false
		}
	}
}

func (b *TradingBot) waitForOrderClosed(req *OrderRequest) bool {
	timeout := time.After(req.CloseTimeout)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			data, err := b.broker.GetOrder(req.Order.ID)
			if err != nil {
				continue
			}
			var updatedOrder Order
			if err := json.Unmarshal(data, &updatedOrder); err != nil {
				continue
			}
			if updatedOrder.IsClosed {
				req.Order.Lock()
				req.Order.Replace(&updatedOrder)
				req.Order.Unlock()
				return true
			}
		case <-timeout:
			return false
		}
	}
}

func (b *TradingBot) cancelOrderWithRetry(req *OrderRequest) bool {
	timeout := time.After(5 * time.Minute)
	for {
		_, err := b.broker.CancelOrder(req.Order.ID)
		if err == nil {
			return true
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-timeout:
			return false
		}
	}
}
