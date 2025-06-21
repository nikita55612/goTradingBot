package trading

import (
	"encoding/json"
	"os"
)

const DefaultTradingBotConfigPath = "./config.json"

type StrategyConfig struct {
	Symbol           string    `json:"symbol"`
	Interval         string    `json:"interval"`
	AvailableBalance float64   `json:"availableBalance"`
	LongRatio        *float64  `json:"longRatio"`
	MartngaleRatios  []float64 `json:"martngaleRatios"`
	TrendZoneFilter  *float64  `json:"trendZoneFilter"`
	LimitOrderOffset *float64  `json:"limitOrderOffset"`
}

type TradingBotConfig struct {
	Strategies []StrategyConfig `json:"strategies"`
}

func DefaultTradingBotConfig() *TradingBotConfig {
	longRatio := .5
	trendZoneFilter := .5
	limitOrderOffset := .01

	sc := StrategyConfig{
		Interval:         "M5",
		AvailableBalance: 15,
		LongRatio:        &longRatio,
		MartngaleRatios:  []float64{1.1},
		TrendZoneFilter:  &trendZoneFilter,
		LimitOrderOffset: &limitOrderOffset,
	}

	return &TradingBotConfig{
		Strategies: []StrategyConfig{sc},
	}
}

func LoadTradingBotConfig(path string) (*TradingBotConfig, error) {
	if path == "" {
		path = DefaultTradingBotConfigPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tradingBotConfig TradingBotConfig
	if err := json.Unmarshal(data, &tradingBotConfig); err != nil {
		return nil, err
	}

	return &tradingBotConfig, nil
}
