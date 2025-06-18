package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit"
	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/trading"
	"github.com/nikita55612/goTradingBot/internal/trading/predict/pyapp"
	"github.com/nikita55612/goTradingBot/internal/trading/strategies"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	defer stop()

	pyapp.SetContext(ctx)
	pyapp.Run()

	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))

	tb := trading.NewTradingBot(ctx, cli.BrokerImpl())

	strategy1 := strategies.NewTrendStrategy(
		"BTCUSDT",
		cdl.M5,
		0,
		0,
		0,
	)

	strategy2 := strategies.NewTrendStrategy(
		"HYPEUSDT",
		cdl.M5,
		0,
		0,
		0,
	)

	_, err := tb.AddStrategy(strategy1)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = tb.AddStrategy(strategy2)
	if err != nil {
		fmt.Println(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Second)
}
