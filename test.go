package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"runtime"
	"time"
	//_ "github.com/lib/pq"
)

var FavoriteStrategyStorage map[string]FavoriteStrategies

var envMinCnt uint

var envMaxLoss float64

var globalMaxSpeed = 0.0
var globalMaxWallet = 0.0
var globalMaxSafety = 0.0

var stat1 = make([]int, len(BarTypes), len(BarTypes))
var stat2 = make([]int, len(IndicatorTypes), len(IndicatorTypes))

type FavoriteStrategies struct {
	Pair                string
	StrategiesMaxWallet []Strategy
	StrategiesMaxSpeed  []Strategy
	StrategiesMaxSafety []Strategy
	TotalStrategies     []Strategy
	CandleData          *CandleData
}

var TestBarTypes = []BarType{
	/*L,*/ /*O, */ C, H, LO, LC, LH, OC, OH, CH, LOC, LOH, LCH, OCH,
}

var TestIndicatorTypes = []IndicatorType{
	IndicatorTypeSma, IndicatorTypeEma, IndicatorTypeDema, IndicatorTypeTema, IndicatorTypeTemaZero, IndicatorType2Ema,
	IndicatorType3Ema, IndicatorTypeEmaTema, IndicatorType2EmaTema, IndicatorType3EmaTema, IndicatorType2Tema,
}

func initTestData(pair string) *FavoriteStrategies {
	data := FavoriteStrategyStorage[pair]
	data.Pair = pair
	return &data
}

func getTestData(pair string) *FavoriteStrategies {
	data, ok := FavoriteStrategyStorage[pair]
	if ok == false {
		return initTestData(pair)
	}
	return &data
}

func (testData *FavoriteStrategies) restore() bool {
	fileName := testData.getFileName()
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&testData)

	return true
}

func (testData *FavoriteStrategies) backup() {
	testData.StrategiesMaxSafety = testData.StrategiesMaxSafety[maxInt(len(testData.StrategiesMaxSafety)-50, 0):]
	testData.StrategiesMaxWallet = testData.StrategiesMaxWallet[maxInt(len(testData.StrategiesMaxWallet)-50, 0):]
	testData.StrategiesMaxSpeed = testData.StrategiesMaxSpeed[maxInt(len(testData.StrategiesMaxSpeed)-100, 0):]
	dataOut := EncodeToBytes(testData)
	_ = ioutil.WriteFile(testData.getFileName(), dataOut, 0644)
}

func (testData *FavoriteStrategies) getFileName() string {
	return fmt.Sprintf("%s_tests_%s_%s.dat", exchange, testData.Pair, resolution)
}

func (testData *FavoriteStrategies) saveToStorage() {
	FavoriteStrategyStorage[testData.Pair] = *testData
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func (candleData *CandleData) getMonthIndex(month int) int {
	monthAgo := time.Now().AddDate(0, -month, 0)

	for i, t := range candleData.Time {
		if t.After(monthAgo) {
			return i
		}
	}

	return candleData.index()
}

func (candleData *CandleData) testPair() {
	testData := getTestData(candleData.Pair)

	proc := runtime.GOMAXPROCS(0)
	tasks := make(chan Strategy, 84)
	ready := make(chan bool, proc)

	oneMonthAgoIndex := candleData.getMonthIndex(1)
	twoMonthsAgoIndex := candleData.getMonthIndex(2)
	threeMonthsAgoIndex := candleData.getMonthIndex(3)
	monthStartIndexes := [3]int{oneMonthAgoIndex, twoMonthsAgoIndex, threeMonthsAgoIndex}
	maxTimeIndex := candleData.index()

	for i := 0; i < proc; i++ {
		go func(tasks <-chan Strategy, ready chan<- bool) {
			for strategy := range tasks {
				candleData.testStrategies(strategy, testData, monthStartIndexes, maxTimeIndex)
			}
			ready <- true
		}(tasks, ready)
	}

	for _, barType1 := range TestBarTypes {
		for _, indicatorType1 := range TestIndicatorTypes {
			for coef1 := range candleData.Indicators[indicatorType1] {
				for _, barType2 := range TestBarTypes {
					for _, indicatorType2 := range TestIndicatorTypes {
						for coef2 := range candleData.Indicators[indicatorType2] {
							for op := 0; op < 60; op += 4 {
								tasks <- Strategy{
									candleData.Pair,
									op,
									Indicator{indicatorType1, barType1, coef1},
									-9999,
									Indicator{indicatorType2, barType2, coef2},
								}
							}
						}
					}
				}
			}
		}
	}
	close(tasks)

	for i := 0; i < proc; i++ {
		<-ready
	}
	close(ready)

	fmt.Printf("\n%+v", stat1)
	fmt.Printf("\n%+v", stat2)
	testData.backup()
}

func (candleData *CandleData) strategyHasEnoughOpens(strategy Strategy, monthIndex, maxTimeIndex int) bool {
	cnt := 0

	for i := monthIndex; i < maxTimeIndex; i++ {
		if 10000*candleData.getIndicatorRatio(strategy, i-1) >= float64(10000+strategy.Op) {
			cnt++
			if cnt == 3 {
				return true
			}
			i += 12
		}
	}

	return false
}

func (candleData *CandleData) testStrategies(strategy Strategy, testData *FavoriteStrategies, monthStartIndexes [3]int, maxTimeIndex int) {
	if false == candleData.strategyHasEnoughOpens(strategy, monthStartIndexes[1], maxTimeIndex) {
		return
	}

	for cl := 20; cl < 500; cl += 20 {
		strategy.Cl = cl
		candleData.testStrategy(strategy, testData, monthStartIndexes, maxTimeIndex)
	}
}

func (candleData *CandleData) testStrategy(strategy Strategy, testData *FavoriteStrategies, monthCntParams [3]int, maxTimeIndex int) {
	wallet, maxWallet := StartDeposit, StartDeposit
	maxLoss, openedCnt, speed, openedPrice := 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum := 0, 0
	cnt, saveStrategy := uint(0), uint(0)
	monthsCnt := make([]uint, len(monthCntParams), len(monthCntParams))

	for i := 1; i < maxTimeIndex; i++ {
		o := candleData.Candles[O][i]
		if openedCnt == 0 {
			if 10000*candleData.getIndicatorRatio(strategy, i-1) >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i

				for mi, monthCnt := range monthCntParams {
					if i >= monthCnt {
						monthsCnt[mi]++
						break
					}
				}
			}
		} else {
			//if 10000*openedPrice/o >= float64(10000+strategy.Cl) {
			//	wallet += (2*openedPrice - o) * openedCnt * Commission
			if 10000*o/openedPrice >= float64(10000+strategy.Cl) {
				wallet += o * openedCnt * Commission

				if wallet > maxWallet {
					maxWallet = wallet
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
			}
		}

		if openedCnt != 0 {
			l := candleData.Candles[L][i]
			loss := 1 - l*openedCnt/maxWallet
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return
				}
			}
		}

	}

	if openedCnt >= 1 {
		wallet += openedPrice * openedCnt
	}

	if rnSum == 0 || cnt < envMinCnt {
		return
	}

	if wallet > globalMaxWallet {
		globalMaxWallet = wallet
		saveStrategy += 1
	}

	speed = (wallet - StartDeposit) / float64(rnSum)
	if speed > globalMaxSpeed*0.996 /* 1000.0*/ {
		saveStrategy += 2
		if speed > globalMaxSpeed {
			globalMaxSpeed = speed
		}
	}

	safety := wallet / maxLoss
	if safety > globalMaxSafety*0.990 {
		saveStrategy += 4
		if safety > globalMaxSafety {
			globalMaxSafety = safety
		}
	}

	if saveStrategy > 0 {
		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
		} else if saveStrategy&2 == 2 {
			testData.StrategiesMaxSpeed = append(testData.StrategiesMaxSpeed, strategy)
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		if (wallet-StartDeposit)/StartDeposit >= 0.2 {
			stat1[strategy.Ind1.BarType]++
			stat1[strategy.Ind2.BarType]++
			stat2[strategy.Ind1.IndicatorType-1]++
			stat2[strategy.Ind2.IndicatorType-1]++
		}

		fmt.Printf("\n %d %s %s %s %s %s %s %s %+v %+v",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
			color.New(color.BgBlue, color.FgYellow).Sprintf("%+v", monthsCnt),
			stat1,
			stat2,
		)
	}
}
