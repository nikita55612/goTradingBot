package predict

import (
	"fmt"
	"slices"
	"sync"

	"github.com/nikita55612/goTradingBot/internal/pkg/cdl"
	"github.com/nikita55612/goTradingBot/internal/trading/predict/pyapp"
	"github.com/nikita55612/goTradingBot/internal/utils/norm"
	"github.com/nikita55612/goTradingBot/internal/utils/numeric"
)

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

type TrendPredictor struct {
	interval    cdl.Interval
	model1      string
	model2      string
	model3      string
	trendZone   []cdl.Candle
	trendBuffer []float64
	ltzfBuffer  []float64
	stzfBuffer  []float64
	lastUpdTime int64
}

func NewTrendPredictor(interval cdl.Interval) *TrendPredictor {
	intervalString := interval.AsString()
	return &TrendPredictor{
		interval:    interval,
		model1:      "PT-" + intervalString,
		model2:      "LNTZS-" + intervalString,
		model3:      "SNTZS-" + intervalString,
		trendBuffer: make([]float64, 0, tpTBS),
	}
}

func (p *TrendPredictor) appendTrendBuffer(values ...float64) int {
	p.trendBuffer = append(p.trendBuffer, values...)

	n := len(p.trendBuffer)
	if n > tpTBS*2 {
		newBuffer := make([]float64, tpTBS)
		copy(newBuffer, p.trendBuffer[tpTBS:])
		p.trendBuffer = newBuffer
	}

	return len(p.trendBuffer)
}

func (p *TrendPredictor) updateTrendZone(candles []cdl.Candle) {
	n := len(candles)
	np := len(p.trendBuffer)

	var trendLen int
	for i := np - 2; i > 0; i-- {
		if (p.trendBuffer[i-1] > .5) != (p.trendBuffer[i] > .5) {
			trendLen = np - i
			break
		}
	}
	p.trendZone = make([]cdl.Candle, trendLen)
	copy(p.trendZone, candles[n-trendLen:])
}

func (p *TrendPredictor) Init(candles []cdl.Candle) error {
	n := len(candles)
	if n < TpIBS {
		return fmt.Errorf("not enough data to initialize: %d < %d", n, TpIBS)
	}

	candles = candles[n-TpIBS:]
	trendFeatures := p.genTrendFeatures(candles)
	trendPreds, err := pyapp.GetPrediction(trendFeatures, p.model1).Unwrap()
	if err != nil {
		return err
	}
	fParts := p.genTZoneFeatures(candles, trendPreds)
	p.ltzfBuffer = fParts[0]
	p.stzfBuffer = fParts[1]

	p.appendTrendBuffer(trendPreds...)
	p.updateTrendZone(candles)
	p.lastUpdTime = candles[len(candles)-1].Time +
		int64(p.interval.AsMilli())

	return nil
}

func (p *TrendPredictor) GetNextPrediction(candles []cdl.Candle) ([2]float64, error) {
	var prediction [2]float64
	n := len(candles)
	if n < tNP+tLB {
		err := fmt.Errorf("not enough candles to predict: %d < %d", n, tNP+tLB)
		return prediction, err
	}

	newTime := candles[n-1].Time
	missCount := int(newTime-p.lastUpdTime+10) / p.interval.AsMilli()
	if missCount <= 0 {
		err := fmt.Errorf("candles data not updated")
		return prediction, err
	}
	if n < tNP+tLB+missCount {
		err := fmt.Errorf("not enough candles: %d < %d", n, tNP+tLB+missCount)
		return prediction, err
	}

	trendFeatures := p.genTrendFeatures(candles[n-tNP-tLB-missCount:])
	trendFeatures = trendFeatures[len(trendFeatures)-missCount:]
	trendPreds, err := pyapp.GetPrediction(trendFeatures, p.model1).Unwrap()
	if err != nil {
		return prediction, err
	}
	if len(trendPreds) == 0 {
		err := fmt.Errorf("received empty prediction")
		return prediction, err
	}

	p.appendTrendBuffer(trendPreds...)
	if missCount == 1 {
		p.trendZone = append(p.trendZone, candles[len(candles)-1])
	} else {
		p.updateTrendZone(candles)
	}

	np := len(p.trendBuffer)
	if (p.trendBuffer[np-2] > .5) != (p.trendBuffer[np-1] > .5) {
		f := p.genNextTZoneFeatures(p.trendZone)
		var model string
		features := make([][]float64, 1)
		if p.trendBuffer[np-2] > .5 {
			copy(p.ltzfBuffer, p.ltzfBuffer[tzNF:])
			copy(p.ltzfBuffer[tzNF*(tzLB-1):], f)
			features[0] = p.stzfBuffer
			model = p.model3
		} else {
			copy(p.stzfBuffer, p.stzfBuffer[tzNF:])
			copy(p.stzfBuffer[tzNF*(tzLB-1):], f)
			features[0] = p.ltzfBuffer
			model = p.model2
		}
		pred, err := pyapp.GetPrediction(features, model).Unwrap()
		if err == nil || len(pred) != 0 {
			prediction[1] = pred[0]
		}
		p.trendZone = []cdl.Candle{candles[len(candles)-1]}
	}
	p.lastUpdTime = newTime
	prediction[0] = p.trendBuffer[np-1]

	return prediction, nil
}

func (p *TrendPredictor) genTrendFeatures(candles []cdl.Candle) [][]float64 {
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

func (p *TrendPredictor) genNextTZoneFeatures(candles []cdl.Candle) []float64 {
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

func (p *TrendPredictor) genTZoneFeatures(candles []cdl.Candle, trend []float64) [2][]float64 {
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
		f := p.genNextTZoneFeatures(candles[st : i+1])
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

	return [2][]float64{lf[len(lf)-1], sf[len(sf)-1]}
}
