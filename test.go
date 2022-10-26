package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"os"
	"runtime"
	"strings"
	"time"
	//_ "github.com/lib/pq"
)

var FavoriteStrategyStorage map[string]FavoriteStrategies

var envLastMonthCnt uint
var envMinCnt uint

var envMaxLoss float64

var globalMaxSpeed = 0.0
var globalMaxWallet = 0.0
var globalMaxSafety = 0.0
var tpMin, tpMax, tpDiv, tpDif int

//var stat1 = make([]int, len(BarTypes), len(BarTypes))
//var stat2 = make([]int, len(IndicatorTypes), len(IndicatorTypes))

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
	testData.StrategiesMaxSafety = testData.StrategiesMaxSafety[maxInt(len(testData.StrategiesMaxSafety)-200, 0):]
	testData.StrategiesMaxWallet = testData.StrategiesMaxWallet[maxInt(len(testData.StrategiesMaxWallet)-10, 0):]
	testData.StrategiesMaxSpeed = testData.StrategiesMaxSpeed[maxInt(len(testData.StrategiesMaxSpeed)-100, 0):]
	dataOut := EncodeToBytes(testData)
	_ = os.WriteFile(testData.getFileName(), dataOut, 0644)
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

type funTestStrategyType func(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int)

func (strategyType StrategyType) getFunction(candleData *CandleData) funTestStrategyType {
	return map[StrategyType]funTestStrategyType{
		Long:    candleData.testLongStrategies,
		Short:   candleData.testShortStrategies,
		LongSl:  candleData.testLongSlStrategies,
		ShortSl: candleData.testShortSlStrategies,
	}[strategyType]
}

func (candleData *CandleData) testPair() {
	envTp := os.Getenv("tp")
	tps := strings.Split(envTp, ";")
	tpMin = int(s2i(tps[0]))
	tpMax = int(s2i(tps[1]))
	tpDiv = int(s2i(tps[2]))
	tpDif = (tpMax - tpMin) / tpDiv
	if tpDif < 1 {
		tpDif = 1
	}

	strategyType := NoStrategyType.value(os.Getenv("strategy_type"))
	strategyFun := strategyType.getFunction(candleData)
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
				ind1 := candleData.getIndicatorValue(strategy.Ind1)
				ind2 := candleData.getIndicatorValue(strategy.Ind2)
				if candleData.strategyHasEnoughOpens(strategy, ind1, ind2, monthStartIndexes[1], maxTimeIndex) {
					strategyFun(strategy, ind1, ind2, testData, maxTimeIndex)
				}
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
									Pair: candleData.Pair,
									Op:   op,
									Ind1: Indicator{indicatorType1, barType1, coef1},
									Ind2: Indicator{indicatorType2, barType2, coef2},
									Type: strategyType,
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

	//fmt.Printf("\n%+v", stat1)
	//fmt.Printf("\n%+v", stat2)
	testData.backup()
}

func (candleData *CandleData) strategyHasEnoughOpens(strategy Strategy, ind1, ind2 []float64, monthIndex, maxTimeIndex int) bool {
	cnt := uint(0)

	for i := monthIndex; i < maxTimeIndex; i++ {
		if 10000*ind1[i-1]/ind2[i-1] >= float64(10000+strategy.Op) {
			cnt++
			if cnt == envLastMonthCnt {
				return true
			}
			i += 8
		}
	}

	return false
}

func (candleData *CandleData) testLongStrategies(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
	for cl := 20; cl < 500; cl += 20 {
		strategy.Tp = cl
		candleData.testLongStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
	}
}

func (candleData *CandleData) testLongStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
	wallet, maxWallet := StartDeposit, StartDeposit
	maxLoss, openedCnt, speed, openedPrice := 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum := 0, 0
	cnt, saveStrategy := uint(0), uint(0)

	lows := candleData.Candles[L]
	opens := candleData.Candles[O]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i

				//for mi, monthCnt := range monthCntParams {
				//	if i >= monthCnt {
				//		monthsCnt[mi]++
				//		break
				//	}
				//}
			}
		} else {
			if 10000*o/openedPrice >= float64(10000+strategy.Tp) {
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
			l := lows[i]
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

	if cnt < envMinCnt {
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

		//if (wallet-StartDeposit)/StartDeposit >= 0.2 {
		//	stat1[strategy.Ind1.BarType]++
		//	stat1[strategy.Ind2.BarType]++
		//	stat2[strategy.Ind1.IndicatorType-1]++
		//	stat2[strategy.Ind2.IndicatorType-1]++
		//}

		fmt.Printf("\n %d %s %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
			//color.New(color.BgBlue, color.FgYellow).Sprintf("%+v", monthsCnt),
			//stat1,
			//stat2,
		)
	}
}

func (candleData *CandleData) testShortStrategies(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
	for tp := 20; tp < 500; tp += 20 {
		strategy.Tp = tp
		candleData.testShortStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
	}
}

func (candleData *CandleData) testShortStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
	wallet, maxWallet := StartDeposit, StartDeposit
	maxLoss, openedCnt, speed, openedPrice := 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum := 0, 0
	cnt, saveStrategy := uint(0), uint(0)

	highs := candleData.Candles[H]
	opens := candleData.Candles[O]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i
			}
		} else {
			if 10000*openedPrice/o >= float64(10000+strategy.Tp) {
				wallet += (2*openedPrice - o) * openedCnt * Commission

				if wallet > maxWallet {
					maxWallet = wallet
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
			}
		}

		if openedCnt != 0 {
			h := highs[i]
			loss := 1 - (2*openedPrice-h)*openedCnt/maxWallet
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

	if cnt < envMinCnt {
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

		fmt.Printf("\n %d %s %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
			//color.New(color.BgBlue, color.FgYellow).Sprintf("%+v", monthsCnt),
			//stat1,
			//stat2,
		)
	}
}

func (candleData *CandleData) testLongSlStrategies(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
_tp:
	for tp := tpMin; tp <= tpMax; tp += tpDif {
		for sl := tp; sl <= tp*5; sl += tpDif * 5 {
			strategy.Tp = tp
			strategy.Sl = sl
			sign := candleData.testLongSlStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
			if sign == TestNoStopLosses || sign == TestNotEnoughCnt {
				continue _tp
			}
		}
	}
}

func (candleData *CandleData) testLongSlStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) testResultSign {
	slCnt := 0
	wallet, maxWallet := StartDeposit, StartDeposit
	maxLoss, openedCnt, speed, openedPrice, sl, tp := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum := 0, 0
	cnt, saveStrategy := uint(0), uint(0)

	candles := candleData.Candles
	lows := candles[L]
	opens := candles[O]
	highs := candles[H]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i

				sl = float64(10000-strategy.Sl) * openedPrice / 10000
				tp = float64(10000+strategy.Tp) * openedPrice / 10000
			}
		}

		if openedCnt != 0 {
			l := lows[i]

			// --
			if l <= sl {
				slCnt++
				wallet += sl * openedCnt * Commission

				loss := 1 - wallet/maxWallet
				if loss > maxLoss {
					maxLoss = loss
					if maxLoss >= envMaxLoss {
						return TestMaxLoss
					}
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
				continue
			}

			// --
			loss := 1 - l*openedCnt/maxWallet
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return TestMaxLoss
				}
			}

			// --
			h := highs[i]
			if h >= tp {
				wallet += tp * openedCnt * Commission

				if wallet > maxWallet {
					maxWallet = wallet
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
			}
		}

	}

	if openedCnt >= 1 {
		wallet += openedPrice * openedCnt
	}

	if cnt < envMinCnt {
		return TestNotEnoughCnt
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
		fmt.Printf("\n %d %s %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
		)

		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
		} else if saveStrategy&2 == 2 {
			testData.StrategiesMaxSpeed = append(testData.StrategiesMaxSpeed, strategy)
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		if slCnt == 0 {
			return TestNoStopLosses
		}
	}

	return TestMaxLoss
}

func (candleData *CandleData) testShortSlStrategies(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
_tp:
	for tp := tpMin; tp <= tpMax; tp += tpDif {
		for sl := tp; sl <= tp*5; sl += tpDif * 5 {
			strategy.Tp = tp
			strategy.Sl = sl
			sign := candleData.testShortSlStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
			if sign == TestNoStopLosses || sign == TestNotEnoughCnt {
				continue _tp
			}
		}
	}
}

func (candleData *CandleData) testShortSlStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) testResultSign {
	slCnt := 0
	wallet, maxWallet := StartDeposit, StartDeposit
	maxLoss, openedCnt, speed, openedPrice, sl, tp := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
	rnOpen, rnSum := 0, 0
	cnt, saveStrategy := uint(0), uint(0)

	candles := candleData.Candles
	lows := candles[L]
	opens := candles[O]
	highs := candles[H]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] >= float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
				rnOpen = i

				sl = float64(10000+strategy.Sl) * openedPrice / 10000
				tp = float64(10000-strategy.Tp) * openedPrice / 10000
			}
		}

		if openedCnt != 0 {
			h := highs[i]

			// --
			if h >= sl {
				slCnt++
				wallet += (2*openedPrice - sl) * openedCnt * Commission

				loss := 1 - wallet/maxWallet
				if loss > maxLoss {
					maxLoss = loss
					if maxLoss >= envMaxLoss {
						return TestMaxLoss
					}
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
				continue
			}

			// --
			loss := 1 - (2*openedPrice-h)*openedCnt/maxWallet
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return TestMaxLoss
				}
			}

			// --
			l := lows[i]
			if l <= tp {
				wallet += (2*openedPrice - tp) * openedCnt * Commission

				if wallet > maxWallet {
					maxWallet = wallet
				}

				openedCnt = 0.0
				cnt++
				rnSum += i - rnOpen
			}
		}

	}

	if openedCnt >= 1 {
		wallet += openedPrice * openedCnt
	}

	if cnt < envMinCnt {
		return TestNotEnoughCnt
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
		fmt.Printf("\n %d %s %s %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
			color.New(color.FgHiRed).Sprintf("%8.2f", speed),
			strategy.String(),
		)

		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
		} else if saveStrategy&2 == 2 {
			testData.StrategiesMaxSpeed = append(testData.StrategiesMaxSpeed, strategy)
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		if slCnt == 0 {
			return TestNoStopLosses
		}
	}

	return TestMaxLoss
}

type testResultSign uint8

const (
	TestMaxLoss testResultSign = iota + 1
	TestNotEnoughCnt
	TestNoStopLosses
)
