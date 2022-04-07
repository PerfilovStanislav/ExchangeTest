package main

import (
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"time"
	//_ "github.com/lib/pq"
)

type IndicatorType string

const (
	IndicatorTypeSma      IndicatorType = "sma"
	IndicatorTypeEma      IndicatorType = "ema"
	IndicatorTypeDema     IndicatorType = "dema"
	IndicatorTypeTema     IndicatorType = "tema"
	IndicatorTypeTemaZero IndicatorType = "tema_zero"
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeTema,
	IndicatorTypeTemaZero,
}

const (
	IndicatorType2Ema     IndicatorType = "2ema"
	IndicatorType3Ema     IndicatorType = "3ema"
	IndicatorTypeEmaTema  IndicatorType = "ema_tema"
	IndicatorType2EmaTema IndicatorType = "2ema_tema"
	IndicatorType3EmaTema IndicatorType = "3ema_tema"
	IndicatorType2Tema    IndicatorType = "2tema"
)

var AdditionalIndicatorTypes = []IndicatorType{
	IndicatorType2Ema, IndicatorType3Ema, IndicatorTypeEmaTema, IndicatorType2EmaTema, IndicatorType3EmaTema,
	IndicatorType2Tema,
}

type BarType string

const (
	Open  BarType = "O"
	Close BarType = "C"
	High  BarType = "H"
	Low   BarType = "L"
)

var BarTypes = [4]BarType{
	Open, Close, High, Low,
}

var Storage map[string]map[tf.CandleInterval]CandleData

func initStorageData(figi string, interval tf.CandleInterval) *CandleData {
	if _, found := Storage[figi]; false == found {
		Storage[figi] = make(map[tf.CandleInterval]CandleData)
	}
	data := Storage[figi][interval]
	data.Indicators = make(map[IndicatorType]map[int]map[BarType][]float64)
	data.Figi = figi
	data.Interval = interval
	return &data
}

func getStorageData(figi string, interval tf.CandleInterval) *CandleData {
	data := Storage[figi][interval]
	return &data
}

type CandleData struct {
	Time       []time.Time
	Candles    map[BarType][]float64
	Indicators map[IndicatorType]map[int]map[BarType][]float64
	Figi       string
	Interval   tf.CandleInterval
}

func (data *CandleData) save() {
	Storage[data.Figi][data.Interval] = *data
}

func (data *CandleData) len() int {
	return len(data.Time)
}

func (data *CandleData) index() int {
	return data.len() - 1
}

func (data *CandleData) lastTime() time.Time {
	return data.Time[data.index()]
}

func (data *CandleData) upsertCandle(c tf.Candle) bool {
	l := data.index()
	if l >= 0 && data.Time[l].Equal(c.TS) {
		data.Time[l] = c.TS
		data.Candles["O"][l] = c.OpenPrice
		data.Candles["C"][l] = c.ClosePrice
		data.Candles["H"][l] = c.HighPrice
		data.Candles["L"][l] = c.LowPrice
		return false
	} else {
		data.Time = append(data.Time, c.TS)
		data.Candles["O"] = append(data.Candles["O"], c.OpenPrice)
		data.Candles["C"] = append(data.Candles["C"], c.ClosePrice)
		data.Candles["H"] = append(data.Candles["H"], c.HighPrice)
		data.Candles["L"] = append(data.Candles["L"], c.LowPrice)
		return true
	}
}

func (data *CandleData) calculateSma(n, i int, barType BarType) float64 {
	if i >= n {
		return data.getSma(n, i-1, barType) + (data.getCandle(n, i, barType)-data.getCandle(n, i-n, barType))/float64(n)
	} else if i > 0 {
		return (data.getSma(n, i-1, barType)*float64(i) + data.getCandle(n, i, barType)) / float64(i+1)
	}
	return data.getCandle(n, 0, barType)
}

func (data *CandleData) calculateDema(n, i int, barType BarType) float64 {
	return 2*data.getEma(n, i, barType) - data.get2Ema(n, i, barType)
}

func (data *CandleData) calculateTema(n, i int, barType BarType) float64 {
	return 3*(data.getEma(n, i, barType)-data.get2Ema(n, i, barType)) + data.get3Ema(n, i, barType)
}

func (data *CandleData) calculateEma(source, prev funGet, n, i int, barType BarType) float64 {
	if i > 0 {
		return (source(n, i, barType)*float64(n) + float64(100-n)*prev(n, i-1, barType)) * 0.01
	}
	return data.Candles[barType][i]
}

func (data *CandleData) calculate2Tema(n, i int, barType BarType) float64 {
	return 3*(data.getEmaTema(n, i, barType)-data.get2EmaTema(n, i, barType)) + data.get3EmaTema(n, i, barType)
}

func (data *CandleData) calculateTemaZero(n, i int, barType BarType) float64 {
	return 2*data.getTema(n, i, barType) - data.get2Tema(n, i, barType)
}

type funGet func(n, i int, barType BarType) float64
type funEma func(source, prev funGet, n, i int, barType BarType) float64

// GET
func (data *CandleData) getCandle(n, i int, barType BarType) float64 {
	return data.Candles[barType][i]
}

func (data *CandleData) get(indicatorType IndicatorType, fun funGet, n, i int, barType BarType) float64 {
	arr := data.Indicators[indicatorType][n][barType]
	if len(arr) > i {
		return arr[i]
	}

	for k := len(arr); k <= i; k++ {
		arr = append(arr, fun(n, k, barType))
		data.Indicators[indicatorType][n][barType] = arr
	}

	return arr[i]
}

func (data *CandleData) ema(indicatorType IndicatorType, fun funEma, source, prev funGet, n, i int, barType BarType) float64 {
	arr := data.Indicators[indicatorType][n][barType]
	if len(arr) > i {
		return arr[i]
	}

	for k := len(arr); k <= i; k++ {
		arr = append(arr, fun(source, prev, n, k, barType))
		data.Indicators[indicatorType][n][barType] = arr
	}

	return arr[i]
}

func (data *CandleData) getSma(n, i int, barType BarType) float64 {
	return data.get(IndicatorTypeSma, data.calculateSma, n, i, barType)
}

func (data *CandleData) getEma(n, i int, barType BarType) float64 {
	return data.ema(IndicatorTypeEma, data.calculateEma, data.getCandle, data.getEma, n, i, barType)
}

func (data *CandleData) get2Ema(n, i int, barType BarType) float64 {
	return data.ema(IndicatorType2Ema, data.calculateEma, data.getEma, data.get2Ema, n, i, barType)
}

func (data *CandleData) get3Ema(n, i int, barType BarType) float64 {
	return data.ema(IndicatorType3Ema, data.calculateEma, data.get2Ema, data.get3Ema, n, i, barType)
}

func (data *CandleData) getDema(n, i int, barType BarType) float64 {
	return data.get(IndicatorTypeDema, data.calculateDema, n, i, barType)
}

func (data *CandleData) getTema(n, i int, barType BarType) float64 {
	return data.get(IndicatorTypeTema, data.calculateTema, n, i, barType)
}

func (data *CandleData) getEmaTema(n, i int, barType BarType) float64 {
	return data.ema(IndicatorTypeEmaTema, data.calculateEma, data.getTema, data.getEmaTema, n, i, barType)
}

func (data *CandleData) get2EmaTema(n, i int, barType BarType) float64 {
	return data.ema(IndicatorType2EmaTema, data.calculateEma, data.getEmaTema, data.get2EmaTema, n, i, barType)
}

func (data *CandleData) get3EmaTema(n, i int, barType BarType) float64 {
	return data.ema(IndicatorType3EmaTema, data.calculateEma, data.get2EmaTema, data.get3EmaTema, n, i, barType)
}

func (data *CandleData) get2Tema(n, i int, barType BarType) float64 {
	return data.get(IndicatorType2Tema, data.calculate2Tema, n, i, barType)
}

func (data *CandleData) getTemaZero(n, i int, barType BarType) float64 {
	return data.get(IndicatorTypeTemaZero, data.calculateTemaZero, n, i, barType)
}

func fillIndicators(data *CandleData) {
	l := data.index()

	for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
		data.Indicators[indicatorType] = make(map[int]map[BarType][]float64)
	}

	for n := 3; n <= 75; n++ {
		fmt.Println(n)
		for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
			data.Indicators[indicatorType][n] = make(map[BarType][]float64)
		}

		for _, barType := range BarTypes {
			data.getSma(n, l, barType)
			data.getEma(n, l, barType)
			data.getDema(n, l, barType)
			data.getTema(n, l, barType)
			data.getTemaZero(n, l, barType)
		}
	}
}
