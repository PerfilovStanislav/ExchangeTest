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

const (
	IndicatorType2Ema     IndicatorType = "2_ema"
	IndicatorType3Ema     IndicatorType = "3_ema"
	IndicatorTypeEmaTema  IndicatorType = "ema_tema"
	IndicatorType2EmaTema IndicatorType = "2_ema_tema"
	IndicatorType3EmaTema IndicatorType = "3_ema_tema"
	IndicatorType2Tema    IndicatorType = "2tema"
)

var AdditionalIndicatorTypes = []IndicatorType{
	IndicatorType2Ema, IndicatorType3Ema, IndicatorTypeEmaTema, IndicatorType2EmaTema, IndicatorType3EmaTema,
	IndicatorType2Tema,
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

var Storage map[string]map[tf.CandleInterval]CandleData

type CandleData struct {
	Time       []time.Time
	Candles    map[string][]float64
	Indicators map[IndicatorType]map[int]map[string][]float64 // IndicatorTypeSma[N=3 OR coef=0.5]Bars
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

func (data *CandleData) upsertCandle(c *tf.Candle) *CandleData {
	l := data.index()
	if l >= 0 && data.Time[l].Equal(c.TS) {
		data.Time[l] = c.TS
		data.Candles["O"][l] = c.OpenPrice
		data.Candles["C"][l] = c.ClosePrice
		data.Candles["H"][l] = c.HighPrice
		data.Candles["L"][l] = c.LowPrice
	} else {
		data.Time = append(data.Time, c.TS)
		data.Candles["O"] = append(data.Candles["O"], c.OpenPrice)
		data.Candles["C"] = append(data.Candles["C"], c.ClosePrice)
		data.Candles["H"] = append(data.Candles["H"], c.HighPrice)
		data.Candles["L"] = append(data.Candles["L"], c.LowPrice)
	}
	return data
}

func fillIndicators(figi string) {
	data := Storage[figi][tf.CandleInterval1Hour]
	//l := len(data.Time)

	for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
		data.Indicators[indicatorType] = make(map[int]map[string][]float64)
	}

	//fmt.Println("Start: ", IndicatorTypeStas)
	//for n := 5; n <= 70; n += 100 /*n++*/ {
	//	data.Indicators[IndicatorTypeStas][n] = make(map[string][]float64)
	//	for _, barType := range BarTypes {
	//		data.Indicators[IndicatorTypeStas][n][barType], _ = calculateStas(n, l, data.Candles[barType])
	//	}
	//}

	//fmt.Println("Start: ", IndicatorTypeSma)
	//for n := 3; n <= 70; n += 100 /*n++*/ {
	//	data.Indicators[IndicatorTypeSma][n] = make(map[string][]float64)
	//	for _, barType := range BarTypes {
	//		for i := 0; i < l; i++ {
	//			data.CalculateSma(n, i, barType)
	//		}
	//	}
	//}

	fmt.Print("Start: another \n")
	for n := 40; n <= 75; n += 50 {
		for _, indicatorType := range append(IndicatorTypes, AdditionalIndicatorTypes...) {
			data.Indicators[indicatorType][n] = make(map[string][]float64)
		}

		//data.Indicators[IndicatorTypeTema][n] = make(map[string][]float64)
		//data.Indicators[IndicatorTypeTemaZeroLag][n] = make(map[string][]float64)
		for _, barType := range BarTypes {
			//for i := 0; i < l; i++ {
			//	data.addEma(n, i, barType)
			//	data.CalculateDema(n, i, barType)
			//}
			//data.getDema(n, 114, barType)
			data.getTemaZero(n, data.index(), barType)
			//data.getTemaZero(n, data.index(), barType)

			//data.Indicators[IndicatorTypeEma][n][barType], _ = calculateEma(n, l, data.Candles[barType])
			//data.Indicators[IndicatorTypeDema][n][barType], _ = calculateDema(n, l, data.Indicators[IndicatorTypeEma][n][barType])
			//data.Indicators[IndicatorTypeTema][n][barType], _ = calculateTema(n, l, data.Indicators[IndicatorTypeEma][n][barType])
			//data.Indicators[IndicatorTypeTemaZeroLag][n][barType], _ = calculateTemaZeroLag(n, l, data.Indicators[IndicatorTypeTema][n][barType])
		}
	}

	fmt.Println(data)
}

func (data *CandleData) CalculateSma(n, i int, barType string) float64 {
	coef := float64(n)
	source := data.Candles[barType]
	var calc float64

	indicator := data.Indicators[IndicatorTypeSma][n][barType]
	if i >= n {
		calc = indicator[i-1] + (source[i]-source[i-n])/coef
	} else if i != 0 {
		calc = (indicator[i-1]*float64(i) + source[i]) / float64(i+1)
	} else {
		indicator = make([]float64, 0)
		calc = source[0]
	}

	indicator = append(indicator, calc)
	data.Indicators[IndicatorTypeSma][n][barType] = indicator
	return calc
}

func (data *CandleData) calculateDema(n, i int, barType string) float64 {
	return 2*data.getEma(n, i, barType) - data.get2Ema(n, i, barType)
}

func (data *CandleData) calculateTema(n, i int, barType string) float64 {
	return 3*(data.getEma(n, i, barType)-data.get2Ema(n, i, barType)) + data.get3Ema(n, i, barType)
}

func (data *CandleData) calculateEma(source, prev funGet, n, i int, barType string) float64 {
	if i > 0 {
		return (source(n, i, barType)*float64(n) + float64(100-n)*prev(n, i-1, barType)) * 0.01
	}
	return data.Candles[barType][i]
}

func (data *CandleData) calculate2Tema(n, i int, barType string) float64 {
	return 3*(data.getEmaTema(n, i, barType)-data.get2EmaTema(n, i, barType)) + data.get3EmaTema(n, i, barType)
}

func (data *CandleData) calculateTemaZero(n, i int, barType string) float64 {
	return 2*data.getTema(n, i, barType) - data.get2Tema(n, i, barType)
}

type funGet func(n, i int, barType string) float64
type funEma func(source, prev funGet, n, i int, barType string) float64

// GET
func (data *CandleData) getCandle(n, i int, barType string) float64 {
	return data.Candles[barType][i]
}

func (data *CandleData) get(indicatorType IndicatorType, fun funGet, n, i int, barType string) float64 {
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

func (data *CandleData) ema(indicatorType IndicatorType, fun funEma, source, prev funGet, n, i int, barType string) float64 {
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

func (data *CandleData) getEma(n, i int, barType string) float64 {
	return data.ema(IndicatorTypeEma, data.calculateEma, data.getCandle, data.getEma, n, i, barType)
}

func (data *CandleData) get2Ema(n, i int, barType string) float64 {
	return data.ema(IndicatorType2Ema, data.calculateEma, data.getEma, data.get2Ema, n, i, barType)
}

func (data *CandleData) get3Ema(n, i int, barType string) float64 {
	return data.ema(IndicatorType3Ema, data.calculateEma, data.get2Ema, data.get3Ema, n, i, barType)
}

func (data *CandleData) getDema(n, i int, barType string) float64 {
	return data.get(IndicatorTypeDema, data.calculateDema, n, i, barType)
}

func (data *CandleData) getTema(n, i int, barType string) float64 {
	return data.get(IndicatorTypeTema, data.calculateTema, n, i, barType)
}

func (data *CandleData) getEmaTema(n, i int, barType string) float64 {
	return data.ema(IndicatorTypeEmaTema, data.calculateEma, data.getTema, data.getEmaTema, n, i, barType)
}

func (data *CandleData) get2EmaTema(n, i int, barType string) float64 {
	return data.ema(IndicatorType2EmaTema, data.calculateEma, data.getEmaTema, data.get2EmaTema, n, i, barType)
}

func (data *CandleData) get3EmaTema(n, i int, barType string) float64 {
	return data.ema(IndicatorType3EmaTema, data.calculateEma, data.get2EmaTema, data.get3EmaTema, n, i, barType)
}

func (data *CandleData) get2Tema(n, i int, barType string) float64 {
	return data.get(IndicatorType2Tema, data.calculate2Tema, n, i, barType)
}

func (data *CandleData) getTemaZero(n, i int, barType string) float64 {
	return data.get(IndicatorTypeTemaZeroLag, data.calculateTemaZero, n, i, barType)
}

//func (data *CandleData) getTemaZero(n, i int, barType string) float64 {
//	arr := data.Indicators[IndicatorTypeTemaZeroLag][n][barType]
//	if len(arr) > i {
//		return arr[i]
//	}
//
//	for k := len(arr); k <= i; k++ {
//		arr = append(arr, data.calculateTemaZero(n, k, barType))
//		data.Indicators[IndicatorTypeTemaZeroLag][n][barType] = arr
//	}
//
//	return arr[i]
//}

//func calculateEma(n, l int, source []float64) ([]float64, float64) {
//	coef := float64(n) * 0.01
//	calc := make([]float64, 0, l)
//
//	calc = append(calc, source[0])
//	for i := 1; i < l; i++ {
//		calc = append(calc, source[i]*coef+(1.0-coef)*calc[i-1])
//	}
//
//	return calc, calc[l-1]
//}

//func calculateDema(n, l int, ema []float64) ([]float64, float64) {
//	calc := make([]float64, 0, l)
//
//	ema2, _ := calculateEma(n, l, ema)
//
//	calc = append(calc, ema[0])
//	for i := 1; i < l; i++ {
//		calc = append(calc, 2*ema[i]-ema2[i])
//	}
//
//	return calc, calc[l-1]
//}

//func calculateTema(n, l int, ema []float64) ([]float64, float64) {
//	calc := make([]float64, 0, l)
//
//	ema2, _ := calculateEma(n, l, ema)
//	ema3, _ := calculateEma(n, l, ema2)
//
//	calc = append(calc, ema[0])
//	for i := 1; i < l; i++ {
//		calc = append(calc, 3*(ema[i]-ema2[i])+ema3[i])
//	}
//
//	return calc, calc[l-1]
//}

//func calculateTemaZeroLag(n, l int, tema []float64) ([]float64, float64) {
//	calc := make([]float64, 0, l)
//
//	tema2, _ := calculateTema(n, l, tema)
//
//	calc = append(calc, tema[0])
//	for i := 1; i < l; i++ {
//		calc = append(calc, 2*tema[i]-tema2[i])
//	}
//
//	return calc, calc[l-1]
//}

//func calculateStas(n, l int, source []float64) ([]float64, float64) {
//	calc := make([]float64, 0, l)
//
//	sma, smaLast := calculateSma(n, l, source)
//	calc = sma[:4]
//
//	for i := 4; i < l; i++ {
//		m := minInt(i+1, n)
//		diff := make([]float64, m, m)
//		for k := 0; k < m; k++ {
//			diff[k] = math.Abs(source[i-m+k+1] - smaLast)
//		}
//
//		_, smaDiff := calculateSma(m, m, diff)
//
//		sum := 0.0
//		for k := 0; k < m; k++ {
//			if diff[k] > smaDiff {
//				sum += source[i-m+k+1]
//			}
//		}
//		calc = append(calc, smaLast-sum/float64(m))
//	}
//
//	return calc, calc[l-1]
//}

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
