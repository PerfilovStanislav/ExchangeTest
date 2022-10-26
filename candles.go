package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/fatih/color"
	"os"
	"reflect"
	"time"
)

type Candle struct {
	L   float64
	O   float64
	C   float64
	H   float64
	LO  float64
	LC  float64
	LH  float64
	OC  float64
	OH  float64
	CH  float64
	LOC float64
	LOH float64
	LCH float64
	OCH float64
	T   time.Time
}

type CandleData struct {
	Pair       string
	Time       []time.Time
	Candles    map[BarType][]float64
	Indicators map[IndicatorType]map[int]map[BarType][]float64
}

var CandleStorage map[string]CandleData

type BarType int8

const (
	L BarType = iota
	O
	C
	H
	LO
	LC
	LH
	OC
	OH
	CH
	LOC
	LOH
	LCH
	OCH
)

func (barType BarType) String() string {
	return map[BarType]string{
		L:   "L",
		O:   "O",
		C:   "C",
		H:   "H",
		LO:  "LO",
		LC:  "LC",
		LH:  "LH",
		OC:  "OC",
		OH:  "OH",
		CH:  "CH",
		LOC: "LOC",
		LOH: "LOH",
		LCH: "LCH",
		OCH: "OCH",
	}[barType]
}

func (barType BarType) value(s string) BarType {
	return map[string]BarType{
		"L":   L,
		"O":   O,
		"C":   C,
		"H":   H,
		"LO":  LO,
		"LC":  LC,
		"LH":  LH,
		"OC":  OC,
		"OH":  OH,
		"CH":  CH,
		"LOC": LOC,
		"LOH": LOH,
		"LCH": LCH,
		"OCH": OCH,
	}[s]
}

var BarTypes = [14]BarType{
	L, O, C, H, LO, LC, LH, OC, OH, CH, LOC, LOH, LCH, OCH,
}

type IndicatorType int8

const (
	IndicatorTypeSma IndicatorType = iota + 1
	IndicatorTypeEma
	IndicatorTypeDema
	IndicatorTypeTema
	IndicatorTypeTemaZero
	IndicatorType2Ema
	IndicatorType3Ema
	IndicatorTypeEmaTema
	IndicatorType2EmaTema
	IndicatorType3EmaTema
	IndicatorType2Tema
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeTema, IndicatorTypeTemaZero, IndicatorType2Ema,
	IndicatorType3Ema, IndicatorTypeEmaTema, IndicatorType2EmaTema, IndicatorType3EmaTema, IndicatorType2Tema,
}

func initCandleData(pair string) *CandleData {
	candleData := CandleStorage[pair]
	candleData.Candles = make(map[BarType][]float64)
	candleData.Indicators = make(map[IndicatorType]map[int]map[BarType][]float64)
	candleData.Pair = pair
	return &candleData
}

func getCandleData(pair string) *CandleData {
	candleData, ok := CandleStorage[pair]
	if ok == false {
		return initCandleData(pair)
	}
	return &candleData
}

func (candleData *CandleData) restore() bool {
	fileName := candleData.getFileName()
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(candleData)
	candleData.save()

	return true
}

func (candleData *CandleData) backup() {
	dataOut := EncodeToBytes(candleData)
	_ = os.WriteFile(candleData.getFileName(), dataOut, 0644)
}

func (candleData *CandleData) getFileName() string {
	return fmt.Sprintf("%s_candles_%s_%s.dat", exchange, candleData.Pair, resolution)
}

func (candleData *CandleData) save() {
	CandleStorage[candleData.Pair] = *candleData
}

func (candleData *CandleData) len() int {
	return len(candleData.Time)
}

func (candleData *CandleData) index() int {
	return candleData.len() - 1
}

func (candleData *CandleData) lastTime() time.Time {
	return candleData.Time[candleData.index()]
}

func (candleData *CandleData) lastCandleValue(barType BarType) float64 {
	return candleData.Candles[barType][candleData.index()]
}

func (candleData *CandleData) upsertCandle(c Candle) bool {
	l := candleData.index()
	if l >= 0 && candleData.Time[l].Equal(c.T) {
		candleData.Time[l] = c.T
		for _, barType := range BarTypes {
			candleData.Candles[barType][l] = c.getPrice(barType)
		}
		return false
	} else {
		candleData.Time = append(candleData.Time, c.T)
		for _, barType := range BarTypes {
			candleData.Candles[barType] = append(candleData.Candles[barType], c.getPrice(barType))
		}
		return true
	}
}

func (candle Candle) getPrice(barType BarType) float64 {
	r := reflect.ValueOf(candle)
	f := reflect.Indirect(r).FieldByName(barType.String())
	return f.Float()
}

func (candleData *CandleData) calculateSma(n, i int, barType BarType) float64 {
	if i >= n {
		return candleData.getSma(n, i-1, barType) + (candleData.getCandle(n, i, barType)-candleData.getCandle(n, i-n, barType))/float64(n)
	} else if i > 0 {
		return (candleData.getSma(n, i-1, barType)*float64(i) + candleData.getCandle(n, i, barType)) / float64(i+1)
	}
	return candleData.getCandle(n, 0, barType)
}

func (candleData *CandleData) calculateDema(n, i int, barType BarType) float64 {
	return 2*candleData.getEma(n, i, barType) - candleData.get2Ema(n, i, barType)
}

func (candleData *CandleData) calculateTema(n, i int, barType BarType) float64 {
	return 3*(candleData.getEma(n, i, barType)-candleData.get2Ema(n, i, barType)) + candleData.get3Ema(n, i, barType)
}

func (candleData *CandleData) calculateEma(source, prev funGet, n, i int, barType BarType) float64 {
	if i > 0 {
		return (source(n, i, barType)*float64(n) + float64(100-n)*prev(n, i-1, barType)) * 0.01
	}
	return candleData.Candles[barType][i]
}

func (candleData *CandleData) calculate2Tema(n, i int, barType BarType) float64 {
	return 3*(candleData.getEmaTema(n, i, barType)-candleData.get2EmaTema(n, i, barType)) + candleData.get3EmaTema(n, i, barType)
}

func (candleData *CandleData) calculateTemaZero(n, i int, barType BarType) float64 {
	return 2*candleData.getTema(n, i, barType) - candleData.get2Tema(n, i, barType)
}

type funGet func(n, i int, barType BarType) float64
type funEma func(source, prev funGet, n, i int, barType BarType) float64

// GET
func (candleData *CandleData) getCandle(n, i int, barType BarType) float64 {
	return candleData.Candles[barType][i]
}

func (candleData *CandleData) get(indicatorType IndicatorType, fun funGet, n, i int, barType BarType) float64 {
	arr := candleData.Indicators[indicatorType][n][barType]
	if len(arr) > i {
		return arr[i]
	}

	for k := len(arr); k <= i; k++ {
		arr = append(arr, fun(n, k, barType))
		candleData.Indicators[indicatorType][n][barType] = arr
	}

	return arr[i]
}

func (candleData *CandleData) ema(indicatorType IndicatorType, fun funEma, source, prev funGet, n, i int, barType BarType) float64 {
	arr := candleData.Indicators[indicatorType][n][barType]
	if len(arr) > i {
		return arr[i]
	}

	for k := len(arr); k <= i; k++ {
		arr = append(arr, fun(source, prev, n, k, barType))
		candleData.Indicators[indicatorType][n][barType] = arr
	}

	return arr[i]
}

func (candleData *CandleData) getSma(n, i int, barType BarType) float64 {
	return candleData.get(IndicatorTypeSma, candleData.calculateSma, n, i, barType)
}

func (candleData *CandleData) getEma(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorTypeEma, candleData.calculateEma, candleData.getCandle, candleData.getEma, n, i, barType)
}

func (candleData *CandleData) get2Ema(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorType2Ema, candleData.calculateEma, candleData.getEma, candleData.get2Ema, n, i, barType)
}

func (candleData *CandleData) get3Ema(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorType3Ema, candleData.calculateEma, candleData.get2Ema, candleData.get3Ema, n, i, barType)
}

func (candleData *CandleData) getDema(n, i int, barType BarType) float64 {
	return candleData.get(IndicatorTypeDema, candleData.calculateDema, n, i, barType)
}

func (candleData *CandleData) getTema(n, i int, barType BarType) float64 {
	return candleData.get(IndicatorTypeTema, candleData.calculateTema, n, i, barType)
}

func (candleData *CandleData) getEmaTema(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorTypeEmaTema, candleData.calculateEma, candleData.getTema, candleData.getEmaTema, n, i, barType)
}

func (candleData *CandleData) get2EmaTema(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorType2EmaTema, candleData.calculateEma, candleData.getEmaTema, candleData.get2EmaTema, n, i, barType)
}

func (candleData *CandleData) get3EmaTema(n, i int, barType BarType) float64 {
	return candleData.ema(IndicatorType3EmaTema, candleData.calculateEma, candleData.get2EmaTema, candleData.get3EmaTema, n, i, barType)
}

func (candleData *CandleData) get2Tema(n, i int, barType BarType) float64 {
	return candleData.get(IndicatorType2Tema, candleData.calculate2Tema, n, i, barType)
}

func (candleData *CandleData) getTemaZero(n, i int, barType BarType) float64 {
	return candleData.get(IndicatorTypeTemaZero, candleData.calculateTemaZero, n, i, barType)
}

func (candleData *CandleData) fillIndicators() {
	l := candleData.index()

	for _, indicatorType := range IndicatorTypes {
		candleData.Indicators[indicatorType] = make(map[int]map[BarType][]float64)
	}

	for n := 3; n <= 70; n++ {
		fmt.Println(n)
		for _, indicatorType := range IndicatorTypes {
			candleData.Indicators[indicatorType][n] = make(map[BarType][]float64)
		}

		for _, barType := range BarTypes {
			candleData.getSma(n, l, barType)
			candleData.getEma(n, l, barType)
			candleData.getDema(n, l, barType)
			candleData.getTema(n, l, barType)
			candleData.getTemaZero(n, l, barType)
		}
	}
}

func (candleData *CandleData) fillIndicator(l int, ind Indicator) float64 {
	return ind.getValue(candleData, l)
}

type StrategyType int8

const (
	NoStrategyType StrategyType = iota
	Long
	Short
	LongSl
	ShortSl
)

func (strategyType StrategyType) String() string {
	return map[StrategyType]string{
		Long:    "long",
		Short:   "short",
		LongSl:  "long_sl",
		ShortSl: "short_sl",
	}[strategyType]
}

func (strategyType StrategyType) value(s string) StrategyType {
	return map[string]StrategyType{
		"long":     Long,
		"short":    Short,
		"long_sl":  LongSl,
		"short_sl": ShortSl,
	}[s]
}

type Strategy struct {
	Pair string
	Op   int
	Ind1 Indicator
	Tp   int
	Ind2 Indicator
	Sl   int
	Type StrategyType
}

type Indicator struct {
	IndicatorType IndicatorType
	BarType       BarType
	Coef          int
}

func (strategy Strategy) getCandleData() *CandleData {
	return getCandleData(strategy.Pair)
}

func (indicatorType IndicatorType) getFunction(data *CandleData) funGet {
	return map[IndicatorType]funGet{
		IndicatorTypeSma:      data.getSma,
		IndicatorTypeEma:      data.getEma,
		IndicatorTypeDema:     data.getDema,
		IndicatorTypeTema:     data.getTema,
		IndicatorTypeTemaZero: data.getTemaZero,
		IndicatorType2Ema:     data.get2Ema,
		IndicatorType3Ema:     data.get3Ema,
		IndicatorTypeEmaTema:  data.getEmaTema,
		IndicatorType2EmaTema: data.get2EmaTema,
		IndicatorType3EmaTema: data.get3EmaTema,
		IndicatorType2Tema:    data.get2Tema,
	}[indicatorType]
}

func (indicator Indicator) getValue(data *CandleData, i int) float64 {
	return indicator.IndicatorType.getFunction(data)(indicator.Coef, i, indicator.BarType)
}

func showStrategies(strategies []Strategy) string {
	var str string
	for _, strategy := range strategies {
		str += strategy.String()
	}
	return str
}

func (strategy Strategy) String() string {
	return fmt.Sprintf("{ %s %s %s %s %s | %s | %s }",
		color.New(color.BgYellow, color.FgBlack).Sprintf("%s", strategy.Type),
		color.New(color.FgBlue).Sprintf("%s", strategy.Pair),
		color.New(color.BgHiBlue, color.FgBlack).Sprintf("%3d", strategy.Op),
		color.New(color.BgHiGreen, color.FgBlack).Sprintf("%3d", strategy.Tp),
		color.New(color.BgHiRed, color.FgBlack).Sprintf("%3d", strategy.Sl),
		strategy.Ind1.String(),
		strategy.Ind2.String(),
	)
}

func (indicator Indicator) String() string {
	return fmt.Sprintf("%s %s %s",
		color.New(color.FgHiBlue).Sprintf("%2d", indicator.IndicatorType),
		color.New(color.FgHiWhite).Sprintf("%3s", indicator.BarType.String()),
		color.New(color.FgYellow).Sprintf("%2d", indicator.Coef),
	)
}

func (candleData *CandleData) getIndicatorValue(indicator Indicator) []float64 {
	return candleData.Indicators[indicator.IndicatorType][indicator.Coef][indicator.BarType]
}

func (candleData *CandleData) getIndicatorRatio(strategy Strategy, index int) float64 {
	return candleData.getIndicatorValue(strategy.Ind1)[index] / candleData.getIndicatorValue(strategy.Ind2)[index]
}
