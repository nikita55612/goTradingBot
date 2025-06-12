package bybit

import (
	"fmt"
	"strconv"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit/models"
	"github.com/nikita55612/goTradingBot/internal/cdl"
)

// AsLocalInterval преобразует cdl.Interval в локальный формат интервала.
func AsLocalInterval(i cdl.Interval) string {
	switch i {
	case cdl.M1:
		return "1"
	case cdl.M3:
		return "3"
	case cdl.M5:
		return "5"
	case cdl.M15:
		return "15"
	case cdl.M30:
		return "30"
	case cdl.H1:
		return "60"
	case cdl.H2:
		return "120"
	case cdl.H4:
		return "240"
	case cdl.H6:
		return "360"
	case cdl.H12:
		return "720"
	case cdl.D1:
		return "D"
	case cdl.D7:
		return "W"
	case cdl.D30:
		return "M"
	}
	return ""
}

// candleStreamFromRawData преобразует сырые данные свечей из WebSocket в структурированный формат.
func candleStreamFromRawData(d *models.CandleStreamRawData) (*cdl.CandleStreamData, error) {
	if len(d.Data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	data := d.Data[0]
	rawData := [7]string{
		strconv.FormatInt(data.End, 10),
		data.Open,
		data.High,
		data.Low,
		data.Close,
		data.Volume,
		data.Turnover,
	}
	candle, err := cdl.ParseCandleFromRawData(rawData)
	if err != nil {
		return nil, err
	}
	interval, err := cdl.ParseInterval(data.Interval)
	if err != nil {
		return nil, err
	}
	return &cdl.CandleStreamData{
		Interval: interval,
		Confirm:  data.Confirm,
		Candle:   candle,
	}, nil
}

// extractCandleFromRawData преобразует массив сырых свечей в массив структурированных свечей
func extractCandleFromResult(res *models.CandleResult) ([]cdl.Candle, error) {
	candles := make([]cdl.Candle, len(res.List))
	for i, v := range res.List {
		candle, err := cdl.ParseCandleFromRawData(v)
		if err != nil {
			return candles, err
		}
		candles[i] = candle
	}
	return candles, nil
}
