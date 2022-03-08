package main

import (
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"math"
	"time"
	//_ "github.com/lib/pq"
)

type IndicatorType string

const (
	IndicatorTypeStas        IndicatorType = "stas"
	IndicatorTypeSma         IndicatorType = "sma"
	IndicatorTypeEma         IndicatorType = "ema"
	IndicatorTypeDema        IndicatorType = "dema"
	IndicatorTypeTema        IndicatorType = "tema"
	IndicatorTypeTemaZeroLag IndicatorType = "zero"
	//IndicatorTypeAma         IndicatorType = "ama"
)

var IndicatorTypes = []IndicatorType{
	IndicatorTypeStas, IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeTema,
	IndicatorTypeTemaZeroLag,
}

var BarTypes = []string{
	"O", "C", "H", "L",
}

type Bars struct {
	O []float64
	C []float64
	H []float64
	L []float64
}

type CandleIndicatorData struct {
	Time       []time.Time
	Candles    map[string][]float64
	Indicators map[IndicatorType]map[float64]map[string][]float64 // IndicatorTypeSma[N=3 OR coef=0.5]Bars
}

var CandleIndicatorStorage map[string]map[tf.CandleInterval]CandleIndicatorData

func fillIndicators(figi string) {
	v := CandleIndicatorStorage[figi][tf.CandleInterval1Hour]
	l := len(v.Time)

	for _, indicatorType := range IndicatorTypes {
		v.Indicators[indicatorType] = make(map[float64]map[string][]float64)
	}

	fmt.Println("Start: ", IndicatorTypeStas)
	for n := 5; n <= 70; n++ {
		coef := float64(n)
		v.Indicators[IndicatorTypeStas][coef] = make(map[string][]float64)
		for _, barType := range BarTypes {
			v.Indicators[IndicatorTypeStas][coef][barType], _ = calculateStas(n, l, v.Candles[barType])
		}
	}

	fmt.Println("Start: ", IndicatorTypeSma)
	for n := 3; n <= 70; n++ {
		coef := float64(n)
		v.Indicators[IndicatorTypeSma][coef] = make(map[string][]float64)
		for _, barType := range BarTypes {
			v.Indicators[IndicatorTypeSma][coef][barType], _ = calculateSma(n, l, v.Candles[barType])
		}
	}

	fmt.Print("Start: another \n")
	for coef := 0.03; coef <= 0.75; coef += 0.01 {
		v.Indicators[IndicatorTypeEma][coef] = make(map[string][]float64)
		v.Indicators[IndicatorTypeDema][coef] = make(map[string][]float64)
		v.Indicators[IndicatorTypeTema][coef] = make(map[string][]float64)
		v.Indicators[IndicatorTypeTemaZeroLag][coef] = make(map[string][]float64)
		for _, barType := range BarTypes {
			v.Indicators[IndicatorTypeEma][coef][barType], _ = calculateEma(coef, l, v.Candles[barType])
			v.Indicators[IndicatorTypeDema][coef][barType], _ = calculateDema(coef, l, v.Indicators[IndicatorTypeEma][coef][barType])
			v.Indicators[IndicatorTypeTema][coef][barType], _ = calculateTema(coef, l, v.Indicators[IndicatorTypeEma][coef][barType])
			v.Indicators[IndicatorTypeTemaZeroLag][coef][barType], _ = calculateTemaZeroLag(coef, l, v.Indicators[IndicatorTypeTema][coef][barType])
		}
	}

	//fmt.Println(v)
}

func calculateSma(n, l int, source []float64) ([]float64, float64) {
	coef := float64(n)
	calc := make([]float64, 0, l)

	calc = append(calc, source[0])
	var sum float64

	for i := 1; i < l; i++ {
		sum = 0
		if i >= n {
			sum = (calc[i-1]*coef - source[i-n] + source[i]) / coef
		} else {
			sum = (calc[i-1]*float64(i) + source[i]) / float64(i+1)
		}
		calc = append(calc, sum)
	}

	return calc, calc[l-1]
}

func calculateEma(coef float64, l int, source []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	calc = append(calc, source[0])
	for i := 1; i < l; i++ {
		calc = append(calc, source[i]*coef+(1.0-coef)*calc[i-1])
	}

	return calc, calc[l-1]
}

func calculateDema(coef float64, l int, ema []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	ema2, _ := calculateEma(coef, l, ema)

	calc = append(calc, ema[0])
	for i := 1; i < l; i++ {
		calc = append(calc, 2*ema[i]-ema2[i])
	}

	return calc, calc[l-1]
}

func calculateTema(coef float64, l int, ema []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	ema2, _ := calculateEma(coef, l, ema)
	ema3, _ := calculateEma(coef, l, ema2)

	calc = append(calc, ema[0])
	for i := 1; i < l; i++ {
		calc = append(calc, 3*(ema[i]-ema2[i])+ema3[i])
	}

	return calc, calc[l-1]
}

func calculateTemaZeroLag(coef float64, l int, tema []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	tema2, _ := calculateTema(coef, l, tema)

	calc = append(calc, tema[0])
	for i := 1; i < l; i++ {
		calc = append(calc, 2*tema[i]-tema2[i])
	}

	return calc, calc[l-1]
}

func calculateStas(n, l int, source []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	sma, smaLast := calculateSma(n, l, source)
	calc = sma[:4]

	for i := 4; i < l; i++ {
		m := minInt(i+1, n)
		diff := make([]float64, m, m)
		for k := 0; k < m; k++ {
			diff[k] = math.Abs(source[i-m+k+1] - smaLast)
		}

		_, smaDiff := calculateSma(m, m, diff)

		sum := 0.0
		for k := 0; k < m; k++ {
			if diff[k] > smaDiff {
				sum += source[i-m+k+1]
			}
		}
		calc = append(calc, smaLast-sum/float64(m))
	}

	return calc, calc[l-1]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func calculateAma(coef float64, l int, source []float64) ([]float64, float64) {
	calc := make([]float64, 0, l)

	calc = append(calc, source[0])
	for i := 1; i < l; i++ {
		calc = append(calc, source[i]*coef*coef+(1.0-coef)*calc[i-1])
	}

	return calc, calc[l-1]
}

//func calculateAma(coef float64, index int, cv []float64) float64 {
//	if index == 0 {
//		return cv[0]
//	}
//	return cv[index]*coef*coef + (1.0-coef*coef)*calculateAma(coef, index-1, cv)
//}

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
