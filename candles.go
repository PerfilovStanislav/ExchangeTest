package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"time"
	//_ "github.com/lib/pq"
)

type IndicatorType string

const (
	IndicatorTypeSma      IndicatorType = "sma"
	IndicatorTypeEma      IndicatorType = "ema"
	IndicatorTypeDema     IndicatorType = "dema"
	IndicatorTypeTema     IndicatorType = "tema"
	IndicatorTypeTemaZero IndicatorType = "temaZ" // zero
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

type CandleData struct {
	FigiInterval string
	//Figi       string
	//Interval   tf.CandleInterval
	Time       []time.Time
	Candles    map[BarType][]float64
	Indicators map[IndicatorType]map[int]map[BarType][]float64
}

var CandleStorage map[string]CandleData

func initCandleData(figiInterval string) *CandleData {
	candleData := CandleStorage[figiInterval]
	candleData.Indicators = make(map[IndicatorType]map[int]map[BarType][]float64)
	candleData.FigiInterval = figiInterval
	return &candleData
}

func getCandleData(figiInterval string) *CandleData {
	candleData, ok := CandleStorage[figiInterval]
	if ok == false {
		return initCandleData(figiInterval)
	}
	return &candleData
}

func (candleData *CandleData) restore() bool {
	fileName := fmt.Sprintf("candles_%s.dat", candleData.FigiInterval)
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
	_ = ioutil.WriteFile(fmt.Sprintf("candles_%s.dat", candleData.FigiInterval), dataOut, 0644)
}

func (candleData *CandleData) save() {
	CandleStorage[candleData.FigiInterval] = *candleData
}

//func (candleData *CandleData) getFigiInterval() string {
//	return figiInterval(candleData.Figi, candleData.Interval)
//}

func (candleData *CandleData) len() int {
	return len(candleData.Time)
}

func (candleData *CandleData) index() int {
	return candleData.len() - 1
}

func (candleData *CandleData) lastTime() time.Time {
	return candleData.Time[candleData.index()]
}

func (candleData *CandleData) upsertCandle(c tf.Candle) bool {
	l := candleData.index()
	if l >= 0 && candleData.Time[l].Equal(c.TS) {
		candleData.Time[l] = c.TS
		candleData.Candles["O"][l] = c.OpenPrice
		candleData.Candles["C"][l] = c.ClosePrice
		candleData.Candles["H"][l] = c.HighPrice
		candleData.Candles["L"][l] = c.LowPrice
		return false
	} else {
		candleData.Time = append(candleData.Time, c.TS)
		candleData.Candles["O"] = append(candleData.Candles["O"], c.OpenPrice)
		candleData.Candles["C"] = append(candleData.Candles["C"], c.ClosePrice)
		candleData.Candles["H"] = append(candleData.Candles["H"], c.HighPrice)
		candleData.Candles["L"] = append(candleData.Candles["L"], c.LowPrice)
		return true
	}
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

	for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
		candleData.Indicators[indicatorType] = make(map[int]map[BarType][]float64)
	}

	for n := 3; n <= 70; n++ {
		fmt.Println(n)
		for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
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
