package main_test

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit"
	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/trading/predict"
	"github.com/nikita55612/goTradingBot/internal/trading/predict/pyapp"
	"github.com/nikita55612/goTradingBot/internal/utils/norm"
	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
)

func TestGenTrendFeatures(t *testing.T) {
	// cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	// candles, err := cli.GetCandles("BTCUSDT", cdl.M15, 10000)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// candles = candles[:len(candles)-1]

	// features := predict.GenTrendFeatures(candles)
}

func TestTrendStrategy(t *testing.T) {
	// cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	// ctx := context.Background()
	// bot := trading.NewTradingBot(ctx, cli.BrokerImpl())

	// bot.AddStrategy(strategies.NewTrendStrategy(
	// 	"BTCUSDT",
	// 	cdl.M1,
	// 	100,
	// 	0.5,
	// 	0.001,
	// ))
	// bot.AddStrategy(strategies.NewTrendStrategy(
	// 	"ADAUSDT",
	// 	cdl.M1,
	// 	100,
	// 	0.5,
	// 	0.001,
	// ))
	// bot.AddStrategy(strategies.NewTrendStrategy(
	// 	"LTCUSDT",
	// 	cdl.M1,
	// 	100,
	// 	0.5,
	// 	0.001,
	// ))
	// bot.AddStrategy(strategies.NewTrendStrategy(
	// 	"HYPEUSDT",
	// 	cdl.M1,
	// 	100,
	// 	0.5,
	// 	0.001,
	// ))
	// bot.AddStrategy(strategies.NewTrendStrategy(
	// 	"TONUSDT",
	// 	cdl.M1,
	// 	100,
	// 	0.5,
	// 	0.001,
	// ))
	// time.Sleep(2 * time.Minute)
	// bot.Stop()
}

func TestPyApp(t *testing.T) {
	// pyapp.Run()
	// defer pyapp.Stop()

	// cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	// candles, err := cli.GetCandles("BTCUSDT", cdl.M5, 10000)
	// if err != nil {
	// 	t.Log(err)
	// 	return
	// }

	// trendFeatures := predict.GenTrendFeatures(candles)
	// trendPreds, err := pyapp.GetPrediction(trendFeatures, "PT-M5").Unwrap()
	// if err != nil {
	// 	t.Log(err)
	// 	return
	// }

	// longFeatureSet, shortFeatureSet := predict.GenTrendQualityZonesFeatures(candles, trendPreds)
	// longFeatureSet = longFeatureSet[len(longFeatureSet)-5:]
	// shortFeatureSet = shortFeatureSet[len(shortFeatureSet)-5:]

	// longTrendZonePreds, err := pyapp.GetPrediction(longFeatureSet, "LTQZ-M5").Unwrap()
	// if err != nil {
	// 	t.Log(err)
	// 	return
	// }
	// shortTrendZonePreds, err := pyapp.GetPrediction(shortFeatureSet, "STQZ-M5").Unwrap()
	// if err != nil {
	// 	t.Log(err)
	// 	return
	// }

	// fmt.Println("longTrendZonePreds:", longTrendZonePreds)
	// fmt.Println("shortTrendZonePreds:", shortTrendZonePreds)

}

func TestTT(t *testing.T) {
	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))

	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		candles, _ := cli.GetCandles("BTCUSDT", cdl.M5, 2)
		fmt.Println(candles)
	}
	candles, _ := cli.GetCandles("BTCUSDT", cdl.M5, 101)
	candles = candles[:len(candles)-1]

	n := len(candles)
	fmt.Println(n)

	newTime := candles[n-1].Time - 3
	candles = candles[:n-7]

	n = len(candles)
	fmt.Println(n)

	lastTime := candles[n-1].Time

	fmt.Println(newTime-lastTime, cdl.M5.AsMilli())

	missCount := int(newTime-lastTime+10) / cdl.M5.AsMilli()

	fmt.Println(missCount)

	// a := []int{1, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0}
	// // b := []int{1, 2, 3, 4, 5, 6, 7}
	// var startIdx int
	// for i := len(a) - 2; i > 0; i-- {
	// 	if (a[i-1] > 0) != (a[i] > 0) {
	// 		startIdx = len(a) - i
	// 		break
	// 	}
	// }

	// b := make([]int, startIdx)
	// copy(b, a[len(a)-startIdx:])
	// fmt.Println(b)
	// fmt.Println(a[len(a)-startIdx:])
}

func TestTrendPredictor(t *testing.T) {
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

	cs := cdl.NewCandleSync(ctx, "BTCUSDT", cdl.M5, 500, cli)
	if err := cs.StartSync(); err != nil {
		fmt.Println(err)
		return
	}

	stream := make(chan *cdl.CandleStreamData)
	done := cs.Subscribe(stream)
	defer func() { close(done) }()

	candles := cs.GetCandles(400)

	tp := predict.NewTrendPredictor(cdl.M5)
	if err := tp.Init(candles); err != nil {
		fmt.Println(err)
		return
	}

	for data := range stream {
		if data.Confirm {
			fmt.Println("Confirm:", data.Candle)
			fmt.Println("Buff:", cs.GetCandles(10)[9])
			p, err := tp.GetNextPrediction(cs.GetCandles(100))
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("trend: %f, zone: %f\n", p[0], p[1])
		}
	}
}

func GenTrendQualityZonesFeaturesAndSignals(candles []cdl.Candle, trendPreds []float64) ([][]float64, []float64, [][]float64, []float64) {
	var (
		lookBack    = 5
		maxTrendLen = 14.
		args        = []cdl.CandleArg{
			cdl.Open,
			cdl.High,
			cdl.Low,
			cdl.Close,
			cdl.Volume,
			cdl.Turnover,
		}
		numFeatures = len(args) + 1
	)
	longLabels := []float64{}
	shortLabels := []float64{}
	longFeatureSet := [][]float64{}
	shortFeatureSet := [][]float64{}
	featureBuffer := make([]float64, numFeatures*lookBack)
	var startIdx int
	var winFilled int
	for i := 1; i < len(trendPreds)-1; i++ {
		if (trendPreds[i-1] > 0.5) == (trendPreds[i] > 0.5) {
			continue
		}
		offset := numFeatures * (lookBack - 1)
		if winFilled < lookBack {
			offset = winFilled * numFeatures
		} else {
			tmpBuffer := make([]float64, numFeatures*lookBack)
			copy(tmpBuffer, featureBuffer)
			featureBuffer = make([]float64, numFeatures*lookBack)
			copy(featureBuffer, tmpBuffer[numFeatures:])
		}
		for j, arg := range args {
			s := cdl.ListOfCandleArg(candles[startIdx:i+1], arg)
			featureBuffer[offset+j] = norm.ZScore(s)
		}
		trendDuration := float64(i - startIdx + 1)
		trendScore := min(maxTrendLen, trendDuration) / maxTrendLen
		featureBuffer[offset+numFeatures-1] = trendScore
		winFilled++
		if winFilled >= lookBack {
			if trendPreds[i-1] > 0.5 {
				longFeatureSet = append(longFeatureSet, featureBuffer)
				if candles[startIdx].C < candles[i].C {
					longLabels = append(longLabels, 1)
				} else {
					longLabels = append(longLabels, 0)
				}
			} else {
				shortFeatureSet = append(shortFeatureSet, featureBuffer)
				if candles[startIdx].C > candles[i].C {
					shortLabels = append(shortLabels, 1)
				} else {
					shortLabels = append(shortLabels, 0)
				}
			}
		}
		startIdx = i
	}
	return longFeatureSet[:len(longLabels)-1], longLabels[1:], shortFeatureSet[:len(shortLabels)-1], shortLabels[1:]
}

func genTrendFeatures(candles []cdl.Candle) [][]float64 {
	n := len(candles)
	features := make([][]float64, len(args)*tLB)
	var wg sync.WaitGroup

	for i, arg := range args {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := cdl.ListOfCandleArg(candles, arg)
			normalize := norm.ZScoreNormalize(s, tNP)
			for s := 0; s < tLB; s++ {
				features[i*tLB+s] = normalize[tLB-s : n-s]
			}
		}()
	}
	wg.Wait()

	return numeric.TransposeMatrix(features)
}

func TestT2(t *testing.T) {
	// for i := 5; i >= 0; i-- {
	// 	fmt.Println(i)
	// }

	pyapp.Run()
	defer pyapp.Stop()

	cli := bybit.NewClientFromEnv(bybit.WithCategory("linear"))
	candles, _ := cli.GetCandles("BTCUSDT", cdl.M5, 400)
	candles = candles[:len(candles)-1]
	n := len(candles)
	of := 14

	tp := predict.NewTrendPredictor(cdl.M5)
	tp.Init(candles[:n-of])
	for i := of - 2; i >= 0; i-- {
		_, err := tp.GetNextPrediction(candles[:n-i])
		if err != nil {
			fmt.Println(err)
		}
	}

	// -----

	f := genTrendFeatures(candles)
	trendPreds, err := pyapp.GetPrediction(f, "PT-M5").Unwrap()
	if err != nil {
		fmt.Println(err)
		return
	}
	l, _, s, _ := GenTrendQualityZonesFeaturesAndSignals(candles, trendPreds)

	fmt.Println("LONG")
	for _, v := range l[len(l)-8:] {
		fmt.Println(v)
	}

	fmt.Println("SHORT")
	for _, v := range s[len(s)-8:] {
		fmt.Println(v)
	}

}

// 						    **
//      ***  **** **     ***
// =====--===---==-======--=-==-=====
//        ****  ***        **
// 	               *******   ***

const (
	tNP   = 21
	tLB   = 9
	tzLB  = 5
	tzNF  = 7
	tzMZ  = 14.
	tpTBS = 200
	TpIBS = 300
)

var (
	args = [6]cdl.CandleArg{
		cdl.Open,
		cdl.High,
		cdl.Low,
		cdl.Close,
		cdl.Volume,
		cdl.Turnover,
	}
)

func genNextTZoneFeatures(candles []cdl.Candle) []float64 {
	f := make([]float64, tzNF)

	for j, a := range args {
		s := cdl.ListOfCandleArg(candles, a)
		f[j] = norm.ZScore(s)
	}
	trendDuration := float64(len(candles))
	trendScore := min(tzMZ, trendDuration) / tzMZ
	f[tzNF-1] = trendScore

	return f
}

func genTZoneFeatures(candles []cdl.Candle, trend []float64) [2][][]float64 {
	nt := len(trend)
	candles = candles[len(candles)-nt:]
	lf, sf := [][]float64{}, [][]float64{}
	lfb := make([]float64, tzNF*tzLB)
	sfb := make([]float64, tzNF*tzLB)

	var st int
	for i := 1; i < nt; i++ {
		prevTrend := trend[i-1] > .5
		if prevTrend == (trend[i] > .5) {
			continue
		}
		f := genNextTZoneFeatures(candles[st : i+1])
		if prevTrend {
			copy(lfb, lfb[tzNF:])
			copy(lfb[tzNF*(tzLB-1):], f)
			lf = append(lf, slices.Clone(lfb))
		} else {
			copy(sfb, sfb[tzNF:])
			copy(sfb[tzNF*(tzLB-1):], f)
			sf = append(sf, slices.Clone(sfb))
		}
		st = i
	}

	return [2][][]float64{lf, sf}
}

func genTZoneFeaturesAndLabels(candles []cdl.Candle, trend []float64) ([2][][]float64, [2][]float64) {
	nt := len(trend)
	candles = candles[len(candles)-nt:]
	lf, sf := [][]float64{}, [][]float64{}
	lfL, sfL := []float64{}, []float64{}
	lfb := make([]float64, tzNF*tzLB)
	sfb := make([]float64, tzNF*tzLB)

	var st int
	for i := 1; i < nt; i++ {
		prevTrend := trend[i-1] > .5
		if prevTrend == (trend[i] > .5) {
			continue
		}
		f := genNextTZoneFeatures(candles[st : i+1])
		if prevTrend {
			copy(lfb, lfb[tzNF:])
			copy(lfb[tzNF*(tzLB-1):], f)
			lf = append(lf, slices.Clone(lfb))

			if candles[st].C < candles[i].C {
				lfL = append(lfL, 1)
			} else {
				lfL = append(lfL, 0)
			}
		} else {
			copy(sfb, sfb[tzNF:])
			copy(sfb[tzNF*(tzLB-1):], f)
			sf = append(sf, slices.Clone(sfb))

			if candles[st].C > candles[i].C {
				sfL = append(sfL, 1)
			} else {
				sfL = append(sfL, 0)
			}

		}
		st = i
	}

	return [2][][]float64{lf, sf}, [2][]float64{lfL, sfL}
}

func TestTTZ(t *testing.T) {
	pyapp.Run()
	defer pyapp.Stop()

	cli := bybit.NewClientFromEnv()
	candles, _ := cli.GetCandles("BTCUSDT", cdl.M5, 200)
	candles = candles[:len(candles)-1]

	f := genTrendFeatures(candles)
	trend, err := pyapp.GetPrediction(f, "PT-M5").Unwrap()
	if err != nil {
		fmt.Println(err)
		return
	}

	zf, _ := genTZoneFeaturesAndLabels(candles, trend)
	fmt.Println("Long:", zf[0])
	fmt.Println("Short:", zf[1])
}
