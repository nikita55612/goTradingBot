package predict

import (
	"sync"

	"github.com/nikita55612/goTradingBot/internal/cdl"
	"github.com/nikita55612/goTradingBot/internal/utils/norm"
)

func GenTrendFeatures(candles []cdl.Candle) [][]float64 {
	var (
		period   = 21
		lookBack = 9
		args     = []cdl.CandleArg{
			cdl.Open,
			cdl.High,
			cdl.Low,
			cdl.Close,
			cdl.Volume,
			cdl.Turnover,
		}
	)
	n := len(candles)
	features := make([][]float64, len(args)*lookBack)
	var wg sync.WaitGroup
	for i, arg := range args {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sequence := cdl.ListOfCandleArg(candles, arg)
			normalize := norm.ZScoreNormalize(sequence, period)
			for s := 0; s < lookBack; s++ {
				features[i*lookBack+s] = normalize[lookBack-s : n-s]
			}
		}()
	}
	wg.Wait()
	return features
}

func GetTrendPrediction(candles []cdl.Candle) {

}
