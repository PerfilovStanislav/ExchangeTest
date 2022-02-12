package main

import (
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"time"
	//_ "github.com/lib/pq"
)

type IndicatorType string

const (
	IndicatorTypeSma  IndicatorType = "sma"
	IndicatorTypeEma  IndicatorType = "ema"
	IndicatorTypeDema IndicatorType = "dema"
	IndicatorTypeAma  IndicatorType = "ama"
	IndicatorTypeTema IndicatorType = "tema"
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeAma,
}

type Candle struct {
	Id   int64
	Time time.Time
	O    float64
	C    float64
	H    float64
	L    float64
	V    int
	Rn   int
}

type Bars struct {
	O []float64
	C []float64
	H []float64
	L []float64
}

//type Indicators struct {
//	Bars
//	N    int64
//	Coef float64
//}

type CandleIndicatorData struct {
	Time       []time.Time
	Candles    Bars
	Indicators map[IndicatorType]map[float64]Bars // IndicatorTypeSma[N=3 OR coef=0.5]Bars
}

var CandleIndicatorStorage map[string]map[tf.CandleInterval]CandleIndicatorData

func CandlesMain() {
	fillCandlesFromDb()
	fillIndicators()
}

func fillCandlesFromDb() int {
	var candles []Candle
	sql := "SELECT id, time, o, c, h, l, v, rn FROM candles WHERE stock_interval_id = 1"
	if err = Db.Select(&candles, sql); err != nil {
		log.Panic(err, sql)
	}

	CandleIndicatorStorage = make(map[string]map[tf.CandleInterval]CandleIndicatorData)
	CandleIndicatorStorage["BBG000B9XRY4"] = make(map[tf.CandleInterval]CandleIndicatorData)

	data := CandleIndicatorData{}
	c := &data.Candles

	data.Indicators = make(map[IndicatorType]map[float64]Bars)
	data.Indicators[IndicatorTypeSma] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeEma] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeDema] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeAma] = make(map[float64]Bars)

	for _, row := range candles {
		data.Time = append(data.Time, row.Time)
		c.O = append(c.O, row.O)
		c.C = append(c.C, row.C)
		c.H = append(c.H, row.H)
		c.L = append(c.L, row.L)
	}
	CandleIndicatorStorage["BBG000B9XRY4"][tf.CandleInterval1Hour] = data

	return 0
}

func fillIndicators() {
	v := CandleIndicatorStorage["BBG000B9XRY4"][tf.CandleInterval1Hour]
	l := len(v.Candles.O)

	ind := Bars{}
	ind.O = make([]float64, l, l)
	ind.C = make([]float64, l, l)
	ind.H = make([]float64, l, l)
	ind.L = make([]float64, l, l)

	fmt.Println("Start IndicatorTypeSma")
	for n := 3; n <= 70; n++ {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateSma(n, index, v.Candles.O)
			ind.C[index] = calculateSma(n, index, v.Candles.C)
			ind.H[index] = calculateSma(n, index, v.Candles.H)
			ind.L[index] = calculateSma(n, index, v.Candles.L)
		}
		v.Indicators[IndicatorTypeSma][float64(n)] = ind
	}

	fmt.Println("Start IndicatorTypeEma")
	for coef := 0.03; coef <= 0.68; coef += 0.05 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateEma(coef, index, v.Candles.O)
			ind.C[index] = calculateEma(coef, index, v.Candles.C)
			ind.H[index] = calculateEma(coef, index, v.Candles.H)
			ind.L[index] = calculateEma(coef, index, v.Candles.L)
		}
		v.Indicators[IndicatorTypeEma][coef] = ind
	}

	fmt.Println("Start IndicatorTypeDema")
	for coef := 0.03; coef <= 0.68; coef += 0.05 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].O)
			ind.C[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].C)
			ind.H[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].H)
			ind.L[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].L)
		}
		v.Indicators[IndicatorTypeDema][coef] = ind
	}

	fmt.Println("Start IndicatorTypeAma")
	for coef := 0.03; coef <= 0.68; coef += 0.05 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateAma(coef, index, v.Candles.O)
			ind.C[index] = calculateAma(coef, index, v.Candles.C)
			ind.H[index] = calculateAma(coef, index, v.Candles.H)
			ind.L[index] = calculateAma(coef, index, v.Candles.L)
		}
		v.Indicators[IndicatorTypeAma][coef] = ind
	}
}

func calculateSma(n, index int, cv []float64) float64 {
	if index < n-1 {
		return calculateSma(index+1, index, cv)
	}

	sum := float64(0)
	for i := index; i > index-n; i-- {
		sum += cv[i]
	}

	return sum / float64(n)
}

func calculateEma(coef float64, index int, cv []float64) float64 {
	if index == 0 {
		return cv[0]
	}
	return cv[index]*coef + (1.0-coef)*calculateEma(coef, index-1, cv)
}

func calculateDema(coef float64, index int, cv []float64) float64 {
	if index == 0 {
		return cv[0]
	}
	return 2*cv[index] - calculateEma(coef, index, cv)
}

func calculateAma(coef float64, index int, cv []float64) float64 {
	if index == 0 {
		return cv[0]
	}
	return cv[index]*coef*coef + (1-coef*coef)*calculateAma(coef, index-1, cv)
}

//func (sma Indicators) Calculate(index int64) float64 {
//	if index < sma.N-1 {
//		return 0
//	}
//
//	sum := 0
//	for i := index; i > index-sma.N; i-- {
//		sum += sum.Add(sma.indicator.Calculate(i))
//	}
//
//	result := sum.Div(big.NewFromInt(sma.window))
//
//	return result
//}

//func sma(n int64, ar *[]float64) {
//	sum := big.ZERO
//	for i := index; i > index-sma.window; i-- {
//		sum = sum.Add(sma.indicator.Calculate(i))
//	}
//
//	result := sum.Div(big.NewFromInt(sma.window))
//
//	return result
//}
