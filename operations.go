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
		testSliceOfOperations(0, len(testDataSlice), testDataSlice, []Strategy{}, &maxWallet)
	}
}

func testSliceOfOperations(i, l int, testDataSlice []*TestData, operationParameters []Strategy, maxWallet *float64) {
	testData := testDataSlice[i]
	if i == l-1 {
		parallel(0, len(testData.TotalStrategies), func(ys <-chan int) {
			for y := range ys {
				testOperations(append(operationParameters, testData.TotalStrategies[y]), maxWallet)
			}
		})
		return
	} else {
		for _, op := range testData.TotalStrategies {
			testSliceOfOperations(i+1, l, testDataSlice, append(operationParameters, op), maxWallet)
		}
	}
}

func testOperations(strategies []Strategy, maxWallet *float64) {
	wallet := StartDeposit
	rnOpen := 0
	rnSum := 0
	openedCnt := 0.0
	cnt := 0
	cl := 0
	show := false
	var openedPrice float64

	var candleData *CandleData

	for i, t := range operationTestTimes.totalTimes[1:] {
		if i == 0 {
			continue
		}

		if openedCnt == 0 {
			for _, strategy := range strategies {
				candleData = getCandleData(strategy.Pair)
				index := operationTestTimes.indexes[strategy.Pair][t]
				if index > 0 {
					x := 10000 * candleData.getIndicatorRatio(strategy, index-1)
					if x >= float64(10000+strategy.Op) {
						cl = strategy.Cl

						openedPrice = candleData.Candles[O][index]
						openedCnt = wallet / openedPrice
						wallet -= openedPrice * openedCnt
						rnOpen = index
						break
					}
				}
			}
		} else {
			index := operationTestTimes.indexes[candleData.Pair][t]
			if index > 0 {
				o := candleData.Candles[O][index]
				if 10000*o/openedPrice >= float64(10000+cl) {
					wallet += o * openedCnt * Commission

					cl = 0
					openedCnt = 0
					cnt++
					rnSum += index - rnOpen
				}
			}
		}

	}

	if openedCnt >= 1 {
		wallet += openedPrice * openedCnt
	}

	if wallet > *maxWallet {
		*maxWallet = wallet
		show = true
	}

	if show {
		fmt.Printf("\n %s %s %s %s",
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.BgBlue).Sprintf("cnt:%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			showOperations(strategies),
		)
	}

	show = false
}
