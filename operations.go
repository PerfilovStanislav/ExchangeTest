package main

import (
	"context"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"time"
)

type OperationParameter struct {
	FigiInterval string
	Op           int
	Ind1         IndicatorParameter
	Cl           int
	Ind2         IndicatorParameter
	//Figi       string
	//Interval   tf.CandleInterval
}

type IndicatorParameter struct {
	IndicatorType IndicatorType
	BarType       BarType
	Coef          int
}

var operationsTestMatrix [][]*TestData

var operationTestTimes = struct {
	totalTimes []time.Time
	//exist      map[*CandleData]map[time.Time]bool
	//indexes    map[*CandleData]map[time.Time]int
	exist   map[string]map[time.Time]bool
	indexes map[string]map[time.Time]int
}{}

type TestOperationStorage struct {
	Wallet, openedPrice, speed, maxWallet, maxSpeed float64
}

func testMatrixOperations(operationsTestMatrix [][]*TestData) {
	var maxWallet float64
	for _, testDataSlice := range operationsTestMatrix {
		testSliceOfOperations(0, len(testDataSlice), testDataSlice, []OperationParameter{}, &maxWallet)
	}
}

func testSliceOfOperations(i, l int, testDataSlice []*TestData, operationParameters []OperationParameter, maxWallet *float64) {
	testData := testDataSlice[i]
	if i == l-1 {
		parallel(0, len(testData.TotalOperations), func(ys <-chan int) {
			for y := range ys {
				testOperations(append(operationParameters, testData.TotalOperations[y]), maxWallet)
			}
		})
		return
	} else {
		for _, op := range testData.TotalOperations {
			testSliceOfOperations(i+1, l, testDataSlice, append(operationParameters, op), maxWallet)
		}
	}
}

func testOperations(operationParameters []OperationParameter, maxWallet *float64) {
	wallet := StartDeposit
	rnOpen := 0
	rnSum := 0
	openedCnt := 0
	cnt := 0
	cl := 0
	show := false
	var openedPrice float64

	var candleData *CandleData

	for _, t := range operationTestTimes.totalTimes[1:] {
		//if i == 0 {
		//	continue
		//}

		if openedCnt == 0 {
			for _, parameter := range operationParameters {
				candleData = getCandleData(parameter.FigiInterval)
				index := operationTestTimes.indexes[parameter.FigiInterval][t]
				if index > 0 {
					x := 10000 * candleData.getIndicatorRatio(parameter, index-1)
					if x >= float64(10000+parameter.Op) {
						cl = parameter.Cl

						openedPrice = candleData.Candles["O"][index]
						openedCnt = int(wallet / openedPrice)
						wallet -= openedPrice * float64(openedCnt)
						rnOpen = index
						break
					}
				}
			}
		} else {
			index := operationTestTimes.indexes[candleData.FigiInterval][t]
			if index > 0 {
				o := candleData.Candles["O"][index]
				if 10000*o/openedPrice >= float64(10000+cl) {
					wallet += o * float64(openedCnt)

					cl = 0
					openedCnt = 0
					cnt++
					rnSum += index - rnOpen
				}
			}
		}

	}

	if openedCnt >= 1 {
		wallet += openedPrice * float64(openedCnt) * Commission
	}

	if wallet > *maxWallet {
		*maxWallet = wallet
		show = true
	}

	if show {
		fmt.Printf("\n %s %s %s %s",
			color.New(color.FgHiGreen).Sprintf("%8d", int(wallet-StartDeposit)),
			color.New(color.BgBlue).Sprintf("cnt:%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			showOperations(operationParameters),
		)
	}

	show = false
}

func showOperation(operation OperationParameter) string {
	figi, _ := getFigiAndInterval(operation.FigiInterval)
	return fmt.Sprintf("{%s %d %d|%s|%s}",
		color.New(color.FgHiRed).Sprintf("%s", figi),
		operation.Op,
		operation.Cl,
		showIndicator(operation.Ind1),
		showIndicator(operation.Ind2),
	)
}

func showOperations(operations []OperationParameter) string {
	var str string
	for _, operation := range operations {
		str = str + showOperation(operation)
	}
	return str
}

func showIndicator(indicator IndicatorParameter) string {
	return fmt.Sprintf("%s %s %s",
		color.New(color.FgHiBlue).Sprintf("%d", indicator.IndicatorType),
		color.New(color.FgWhite).Sprint(indicator.BarType),
		color.New(color.FgHiWhite).Sprintf("%d", indicator.Coef),
	)
}

func (indicatorType IndicatorType) getFunction(data *CandleData) funGet {
	switch indicatorType {
	case IndicatorTypeSma:
		return data.getSma
	case IndicatorTypeEma:
		return data.getEma
	case IndicatorTypeDema:
		return data.getDema
	case IndicatorTypeTema:
		return data.getTema
	case IndicatorTypeTemaZero:
		return data.getTemaZero
	}
	return nil
}

func (indicator IndicatorParameter) getValue(data *CandleData, i int) float64 {
	return indicator.IndicatorType.getFunction(data)(indicator.Coef, i, indicator.BarType)
}

//func listenCandles(tinkoff *Tinkoff) {
//	for figi, figiValue := range OperationParameters {
//		for interval, _ := range figiValue {
//			err := tinkoff.StreamClient.SubscribeCandle(figi, interval, requestID())
//			if err != nil {
//				log.Fatalln(err)
//			}
//		}
//	}
//}

//func newCandleEvent(tinkoff *Tinkoff, candle tf.Candle) {
//	data := getCandleData(candle.FIGI, candle.Interval)
//
//	if data.upsertCandle(candle) {
//		for _, parameter := range OperationParameters[candle.FIGI][candle.Interval] {
//			checkOpening(tinkoff, data, candle, parameter)
//		}
//	}
//
//	data.saveToStorage()
//}

func (tinkoff *Tinkoff) Open(figi string, lots int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	placedOrder, err := tinkoff.getApiClient().MarketOrder(ctx, tf.DefaultAccount, figi, lots, tf.BUY)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%+v\n", placedOrder)
}

func (tinkoff *Tinkoff) Close(figi string, lots int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	placedOrder, err := tinkoff.getApiClient().MarketOrder(ctx, tf.DefaultAccount, figi, lots, tf.SELL)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%+v\n", placedOrder)
}

func checkOpening(tinkoff *Tinkoff, data *CandleData, candle tf.Candle, parameter OperationParameter) {
	i := data.index() - 1
	val1 := parameter.Ind1.getValue(data, i)
	val2 := parameter.Ind2.getValue(data, i)
	tinkoff.Open(candle.FIGI, 1)
	if val1*10000/val2 >= float64(10000+parameter.Op) {
		tinkoff.Open(candle.FIGI, 1)
	}
}

//func (parameter OperationParameter) getFigiInterval() string {
//	return figiInterval(parameter.Figi, parameter.Interval)
//}
