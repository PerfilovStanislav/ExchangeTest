package main

import (
	"fmt"
	"github.com/fatih/color"
	"time"
)

var operationsTestMatrix [][]*TestData

var operationTestTimes = struct {
	totalTimes []time.Time
	exist      map[string]map[time.Time]bool
	indexes    map[string]map[time.Time]int
}{}

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

						openedPrice = candleData.Candles[O][index]
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
				o := candleData.Candles[O][index]
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
