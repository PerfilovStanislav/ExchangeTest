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
	IndicatorTypeSma  IndicatorType = "sma"
	IndicatorTypeEma  IndicatorType = "ema"
	IndicatorTypeDema IndicatorType = "dema"
	IndicatorTypeAma  IndicatorType = "ama"
	IndicatorTypeTema IndicatorType = "tema"
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeAma, IndicatorTypeTema,
}

type Bars struct {
	O []float64
	C []float64
	H []float64
	L []float64
}

type CandleIndicatorData struct {
	Time       []time.Time
	Candles    Bars
	Indicators map[IndicatorType]map[float64]Bars // IndicatorTypeSma[N=3 OR coef=0.5]Bars
}

var CandleIndicatorStorage map[string]map[tf.CandleInterval]CandleIndicatorData

func fillIndicators(figi string) {
	v := CandleIndicatorStorage[figi][tf.CandleInterval1Hour]
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
	for coef := 0.03; coef <= 0.7; coef += 0.01 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateEma(coef, index, v.Candles.O)
			ind.C[index] = calculateEma(coef, index, v.Candles.C)
			ind.H[index] = calculateEma(coef, index, v.Candles.H)
			ind.L[index] = calculateEma(coef, index, v.Candles.L)
		}
		v.Indicators[IndicatorTypeEma][coef] = ind
	}

	fmt.Println("Start IndicatorTypeDema")
	for coef := 0.03; coef <= 0.7; coef += 0.01 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].O)
			ind.C[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].C)
			ind.H[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].H)
			ind.L[index] = calculateDema(coef, index, v.Indicators[IndicatorTypeEma][coef].L)
		}
		v.Indicators[IndicatorTypeDema][coef] = ind
	}

	fmt.Println("Start IndicatorTypeAma")
	for coef := 0.03; coef <= 0.7; coef += 0.01 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateAma(coef, index, v.Candles.O)
			ind.C[index] = calculateAma(coef, index, v.Candles.C)
			ind.H[index] = calculateAma(coef, index, v.Candles.H)
			ind.L[index] = calculateAma(coef, index, v.Candles.L)
		}
		v.Indicators[IndicatorTypeAma][coef] = ind
	}

	fmt.Println("Start IndicatorTypeTema")
	for coef := 0.03; coef <= 0.7; coef += 0.01 {
		for index := 0; index < l; index++ {
			ind.O[index] = calculateTema(coef, index, v.Indicators[IndicatorTypeEma][coef].O, v.Indicators[IndicatorTypeDema][coef].O)
			ind.C[index] = calculateTema(coef, index, v.Indicators[IndicatorTypeEma][coef].C, v.Indicators[IndicatorTypeDema][coef].C)
			ind.H[index] = calculateTema(coef, index, v.Indicators[IndicatorTypeEma][coef].H, v.Indicators[IndicatorTypeDema][coef].H)
			ind.L[index] = calculateTema(coef, index, v.Indicators[IndicatorTypeEma][coef].L, v.Indicators[IndicatorTypeDema][coef].L)
		}
		v.Indicators[IndicatorTypeTema][coef] = ind
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

func calculateDema(coef float64, index int, emaV []float64) float64 {
	if index == 0 {
		return emaV[0]
	}
	return 2*emaV[index] - calculateEma(coef, index, emaV)
}

func calculateAma(coef float64, index int, cv []float64) float64 {
	if index == 0 {
		return cv[0]
	}
	return cv[index]*coef*coef + (1-coef*coef)*calculateAma(coef, index-1, cv)
}

func calculateTema(coef float64, index int, emaV []float64, demaV []float64) float64 {
	if index == 0 {
		return 4*emaV[0] - 3*demaV[0]
	}
	return 3*emaV[index] - 3*demaV[index] + calculateEma(coef, index, demaV)
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
