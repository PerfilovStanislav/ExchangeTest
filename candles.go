package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"reflect"
	"time"
	//_ "github.com/lib/pq"
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

type IndicatorType int8

const (
	IndicatorTypeSma      IndicatorType = 1
	IndicatorTypeEma      IndicatorType = 2
	IndicatorTypeDema     IndicatorType = 3
	IndicatorTypeTema     IndicatorType = 4
	IndicatorTypeTemaZero IndicatorType = 5 // zero
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeTema,
	IndicatorTypeTemaZero,
}

const (
	IndicatorType2Ema     IndicatorType = 6
	IndicatorType3Ema     IndicatorType = 7
	IndicatorTypeEmaTema  IndicatorType = 8
	IndicatorType2EmaTema IndicatorType = 9
	IndicatorType3EmaTema IndicatorType = 10
	IndicatorType2Tema    IndicatorType = 11
)

var AdditionalIndicatorTypes = []IndicatorType{
	IndicatorType2Ema, IndicatorType3Ema, IndicatorTypeEmaTema, IndicatorType2EmaTema, IndicatorType3EmaTema,
	IndicatorType2Tema,
}

type BarType int8

const (
	L = BarType(iota)
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

func (barType BarType) getName() string {
	return [14]string{
		"L", "O", "C", "H", "LO", "LC", "LH", "OC", "OH", "CH", "LOC", "LOH", "LCH", "OCH",
	}[barType]
}

var BarTypes = [14]BarType{
	LOC, LOH, LCH, OCH, LO, LC, LH, OC, OH, CH, O, C, H, L,
}

var TestBarTypes = [10]BarType{
	LOC, LOH, LCH, OCH, LO, LC, LH, OC, OH, CH, //O, C, H, L,
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
	f := reflect.Indirect(r).FieldByName(barType.getName())
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
