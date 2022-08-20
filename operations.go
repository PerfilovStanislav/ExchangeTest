package main

import (
	"fmt"
	"github.com/fatih/color"
	"time"
)

var strategiesTestMatrix [][]*FavoriteStrategies

var strategyTestTimes = struct {
	totalTimes []time.Time
	exist      map[string]map[time.Time]bool
	indexes    map[string]map[time.Time]int
}{}

func testMatrixStrategies(strategiesTestMatrix [][]*FavoriteStrategies) {
	var globalMaxWallet = 0.0
	var globalMaxSafety = 0.0

	for _, testDataSlice := range strategiesTestMatrix {
		testSliceOfStrategies(0, len(testDataSlice), testDataSlice, []Strategy{}, &globalMaxWallet, &globalMaxSafety)
	}
}

func testSliceOfStrategies(i, l int, testDataSlice []*FavoriteStrategies, strategies []Strategy, globalMaxWallet, globalMaxSafety *float64) {
	testData := testDataSlice[i]
	if i == l-1 {
		parallel(0, len(testData.TotalStrategies), func(ys <-chan int) {
			for y := range ys {
				testStrategies(append(strategies, testData.TotalStrategies[y]), globalMaxWallet, globalMaxSafety)
			}
		})
		return
	} else {
		for _, op := range testData.TotalStrategies {
			testSliceOfStrategies(i+1, l, testDataSlice, append(strategies, op), globalMaxWallet, globalMaxSafety)
		}
	}
}

func testStrategies(strategies []Strategy, globalMaxWallet, globalMaxSafety *float64) {
	wallet := StartDeposit
	maxWallet := StartDeposit
	rnOpen, rnSum, cnt, cl, saveStrategy := 0, 0, 0, 0, 0
	openedCnt, maxLoss := 0.0, 0.0
	var openedPrice float64

	var candleData *CandleData

	for _, t := range strategyTestTimes.totalTimes[1:] {
		//if i == 0 {
		//	continue
		//}

		if openedCnt == 0 {
			for _, strategy := range strategies {
				candleData = getCandleData(strategy.Pair)
				index := strategyTestTimes.indexes[strategy.Pair][t]
				if index > 0 && 10000*candleData.getIndicatorRatio(strategy, index-1)/float64(10000+strategy.Op) >= 1.0 {
					cl = strategy.Cl

					openedPrice = candleData.Candles[O][index]
					openedCnt = wallet / openedPrice
					wallet -= openedPrice * openedCnt
					rnOpen = index
					break
				}
			}
		} else {
			index := strategyTestTimes.indexes[candleData.Pair][t]
			if index > 0 {
				o := candleData.Candles[O][index]
				if 10000*o/openedPrice >= float64(10000+cl) {
					wallet += o * openedCnt * Commission

					if wallet > maxWallet {
						maxWallet = wallet
					}

					cl = 0
					openedCnt = 0
					cnt++
					rnSum += index - rnOpen
				}
			}
		}

		if openedCnt != 0 {
			index := strategyTestTimes.indexes[candleData.Pair][t]
			l := candleData.Candles[L][index]
			loss := 1 - l*openedCnt/maxWallet
			if loss > 0.18 {
				return
			}
			if loss > maxLoss {
				maxLoss = loss
			}
		}
	}

	if openedCnt >= 1 {
		wallet += openedPrice * openedCnt
	}

	if wallet > *globalMaxWallet {
		*globalMaxWallet = wallet
		saveStrategy += 1
	}

	safety := wallet / maxLoss
	if safety > *globalMaxSafety {
		*globalMaxSafety = safety
		saveStrategy += 4
	}

	if saveStrategy > 0 {
		fmt.Printf("\n %d %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("cnt:%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			showStrategies(strategies),
		)
	}
}
