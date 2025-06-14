package trading

import (
	"context"

	"github.com/nikita55612/goTradingBot/internal/broker"
)

// сигналы
// 1. Запуск
// 2. Пауза

type TradingBot struct {
	ctx     context.Context
	broker  broker.Broker
	subData *SubData
}

func NewTradingBot(ctx context.Context, broker broker.Broker) *TradingBot {

}
