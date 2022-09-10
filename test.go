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
	//_ "github.com/lib/pq"
)

var FavoriteStrategyStorage map[string]FavoriteStrategies

var envMinCnt int

var envMaxLoss float64

const additionalMoney = 0

var globalMaxSpeed = 0.0
var globalMaxWallet = 0.0
var globalMaxSafety = 0.0

type FavoriteStrategies struct {
	Pair                string
	StrategiesMaxWallet []Strategy
	StrategiesMaxSpeed  []Strategy
	StrategiesMaxSafety []Strategy
	TotalStrategies     []Strategy
	CandleData          *CandleData
}

var TestBarTypes = []BarType{
	LOC, LOH, LCH, OCH, LO, LC, LH, OC, OH, CH, //O, C, H, L,
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
	fileName := fmt.Sprintf("tests_%s.dat", testData.Pair)
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&testData)

	return true
}

func (testData *FavoriteStrategies) backup() {
	testData.StrategiesMaxSafety = testData.StrategiesMaxSafety[maxInt(len(testData.StrategiesMaxSafety)-40, 0):]
	testData.StrategiesMaxWallet = testData.StrategiesMaxWallet[maxInt(len(testData.StrategiesMaxWallet)-40, 0):]
	testData.StrategiesMaxSpeed = testData.StrategiesMaxSpeed[maxInt(len(testData.StrategiesMaxSpeed)-100, 0):]
	dataOut := EncodeToBytes(testData)
	_ = ioutil.WriteFile(fmt.Sprintf("tests_%s.dat", testData.Pair), dataOut, 0644)
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

func (candleData *CandleData) testPair() {
	testData := getTestData(candleData.Pair)

	proc := runtime.GOMAXPROCS(0)
	tasks := make(chan Strategy, 2000)
	ready := make(chan bool, proc)

	for i := 0; i < proc; i++ {
		go func(tasks <-chan Strategy, ready chan bool) {
			for strategy := range tasks {
				candleData.testStrategy(strategy, testData)
			}
			ready <- true
		}(tasks, ready)
	}

	for _, barType1 := range TestBarTypes {
		for _, indicatorType1 := range IndicatorTypes {
			for coef1 := range candleData.Indicators[indicatorType1] {
				for _, barType2 := range TestBarTypes {
					for _, indicatorType2 := range IndicatorTypes {
						for coef2 := range candleData.Indicators[indicatorType2] {
							for op := 0; op < 60; op += 4 {
								for cl := 20; cl < 500; cl += 20 {
									strategy := Strategy{
										candleData.Pair,
										op,
										Indicator{indicatorType1, barType1, coef1},
										cl,
										Indicator{indicatorType2, barType2, coef2},
									}
									tasks <- strategy
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
		fmt.Println("XXX")
	}
	close(ready)

	testData.backup()
}

func (candleData *CandleData) testStrategy(strategy Strategy, testData *FavoriteStrategies) {
	wallet := StartDeposit
	maxWallet := StartDeposit
	maxLoss, openedCnt, speed, openedPrice, addedMoney := 0.0, 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum, cnt, saveStrategy := 0, 0, 0, 0

	for i, _ := range candleData.Time {
		wallet += additionalMoney
		addedMoney += additionalMoney
		if i == 0 {
			continue
		}

		o := candleData.Candles[O][i]
		if openedCnt == 0 {
			if 10000*candleData.getIndicatorRatio(strategy, i-1) >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i
			}
		} else {
			if 10000*o/openedPrice >= float64(10000+strategy.Cl) {
				wallet += o * openedCnt * Commission

				if wallet-addedMoney > maxWallet {
					maxWallet = wallet - addedMoney
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
			}
		}

		if openedCnt != 0 {
			l := candleData.Candles[L][i]
			loss := 1 - l*openedCnt/(maxWallet+addedMoney)
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return
				}
			}
		}

	}

	wallet -= addedMoney

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
	if speed > (globalMaxSpeed)*0.996 /* 1000.0*/ {
		saveStrategy += 2
		if speed > globalMaxSpeed {
			globalMaxSpeed = speed
		}
	}

	safety := wallet / maxLoss
	if safety > globalMaxSafety {
		globalMaxSafety = safety
		saveStrategy += 4
	}

	if saveStrategy > 0 {
		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
		} else if saveStrategy&2 == 2 {
			testData.StrategiesMaxSpeed = append(testData.StrategiesMaxSpeed, strategy)
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		fmt.Printf("\n %d %s %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
		)
	}
}
