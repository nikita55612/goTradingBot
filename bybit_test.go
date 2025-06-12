package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit"
	"github.com/nikita55612/goTradingBot/internal/cdl"
)

func TestGetInstrumentInfo(t *testing.T) {
	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	instrumentInfo, err := cli.GetInstrumentInfo("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(instrumentInfo, "", "    ")
	fmt.Println(string(data))
}

func TestGetCandles(t *testing.T) {
	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	candles, err := cli.GetCandles("BTCUSDT", cdl.M15, 10)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(candles, "", "    ")
	fmt.Println(string(data))
}

func TestCandleStream(t *testing.T) {
	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := cli.CandleStream(ctx, "BTCUSDT", cdl.M15)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-time.After(time.Minute)
		cancel()
	}()
	for data := range stream {
		fmt.Printf("%+v\n", data)
	}
}
