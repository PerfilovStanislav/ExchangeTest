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

var envMaxLoss /*, globalMaxSpeed*/, globalMaxWallet, globalMaxSafety float64
var envLastMonthCnt, envMinCnt int
var tpMin, tpMax, tpDiv, tpDif int
var opMin, opMax, opDiv, opDif int

//var stat1 = make([]int, len(BarTypes), len(BarTypes))
//var stat2 = make([]int, len(IndicatorTypes), len(IndicatorTypes))

type FavoriteStrategies struct {
	Pair                string
	StrategiesMaxWallet []Strategy
	//StrategiesMaxSpeed  []Strategy
	StrategiesMaxSafety []Strategy
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

func (strategy Strategy) getTestData() *FavoriteStrategies {
	return getTestData(strategy.Pair)
}

func getTestData(pair string) *FavoriteStrategies {
	data, ok := FavoriteStrategyStorage[pair]
	if ok == false {
		return initTestData(pair)
	}
	return &data
}

func (favoriteStrategies *FavoriteStrategies) restore() bool {
	fileName := favoriteStrategies.getFileName()
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&favoriteStrategies)

	return true
}

func (favoriteStrategies *FavoriteStrategies) backup() {
	favoriteStrategies.StrategiesMaxSafety = favoriteStrategies.StrategiesMaxSafety[maxInt(len(favoriteStrategies.StrategiesMaxSafety)-200, 0):]
	favoriteStrategies.StrategiesMaxWallet = favoriteStrategies.StrategiesMaxWallet[maxInt(len(favoriteStrategies.StrategiesMaxWallet)-10, 0):]
	//favoriteStrategies.StrategiesMaxSpeed = favoriteStrategies.StrategiesMaxSpeed[maxInt(len(favoriteStrategies.StrategiesMaxSpeed)-100, 0):]
	dataOut := EncodeToBytes(favoriteStrategies)
	_ = os.WriteFile(favoriteStrategies.getFileName(), dataOut, 0644)
}

func (favoriteStrategies *FavoriteStrategies) getFileName() string {
	return fmt.Sprintf("%s_tests_%s_%s_%s.dat",
		exchange,
		favoriteStrategies.Pair,
		resolution,
		os.Getenv("strategy_type"),
	)
}

func (favoriteStrategies *FavoriteStrategies) saveToStorage() {
	FavoriteStrategyStorage[favoriteStrategies.Pair] = *favoriteStrategies
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

type funTestPair func(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int)
type funTestStrategy func(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) testResultSign

func (strategyType StrategyType) getTestPairFunction(candleData *CandleData) funTestPair {
	return map[StrategyType]funTestPair{
		Long:    candleData.testLongStrategies,
		Short:   candleData.testShortStrategies,
		LongSl:  candleData.testLongSlStrategies,
		ShortSl: candleData.testShortSlStrategies,
	}[strategyType]
}

func (strategyType StrategyType) getTestStrategyFunction(candleData *CandleData) funTestStrategy {
	return map[StrategyType]funTestStrategy{
		Long:    candleData.testLongStrategy,
		Short:   candleData.testShortStrategy,
		LongSl:  candleData.testLongSlStrategy,
		ShortSl: candleData.testShortSlStrategy,
	}[strategyType]
}

func (candleData *CandleData) testPair() {
	strategyType := NoStrategyType.value(os.Getenv("strategy_type"))
	strategyFun := strategyType.getTestPairFunction(candleData)
	testData := getTestData(candleData.Pair)

	threads := toInt(os.Getenv("threads"))
	proc := runtime.GOMAXPROCS(threads)
	tasks := make(chan Strategy, 84)
	ready := make(chan bool, threads)

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
							for op := opMin; op <= opMax; op += opDif {
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
	var cnt int

	for i := monthIndex; i < maxTimeIndex; i++ {
		if 10000*ind1[i-1]/ind2[i-1] > float64(10000+strategy.Op) {
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
	for tp := tpMin; tp <= tpMax; tp += tpDif {
		strategy.Tp = tp
		candleData.testLongStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
	}
}

func (candleData *CandleData) testLongStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) testResultSign {
	wallet, maxWallet := StartDeposit, StartDeposit
	var maxLoss, openedCnt, openedPrice float64
	var cnt, saveStrategy int

	lows := candleData.Candles[L]
	opens := candleData.Candles[O]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] > float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt

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
			}
		}

		if openedCnt != 0 {
			l := lows[i]
			loss := 1 - l*openedCnt/maxWallet
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return TestMaxLoss
				}
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

	//speed = (wallet - StartDeposit) / float64(rnSum)
	//if speed > globalMaxSpeed*0.996 /* 1000.0*/ {
	//	saveStrategy += 2
	//	if speed > globalMaxSpeed {
	//		globalMaxSpeed = speed
	//	}
	//}

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
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		//if (wallet-StartDeposit)/StartDeposit >= 0.2 {
		//	stat1[strategy.Ind1.BarType]++
		//	stat1[strategy.Ind2.BarType]++
		//	stat2[strategy.Ind1.IndicatorType-1]++
		//	stat2[strategy.Ind2.IndicatorType-1]++
		//}

		fmt.Printf("\n %d %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			strategy.String(),
			//stat1,
			//stat2,
		)
	}

	return TestNoResult
}

func (candleData *CandleData) testShortStrategies(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) {
	for tp := tpMin; tp <= tpMax; tp += tpDif {
		strategy.Tp = tp
		candleData.testShortStrategy(strategy, ind1, ind2, testData, maxTimeIndex)
	}
}

func (candleData *CandleData) testShortStrategy(strategy Strategy, ind1, ind2 []float64, testData *FavoriteStrategies, maxTimeIndex int) testResultSign {
	wallet, maxWallet := StartDeposit, StartDeposit
	var maxLoss, openedCnt, openedPrice float64
	var cnt, saveStrategy int

	highs := candleData.Candles[H]
	opens := candleData.Candles[O]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] > float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt
			}
		} else {
			if 10000*openedPrice/o >= float64(10000+strategy.Tp) {
				wallet += (2*openedPrice - o) * openedCnt * Commission

				if wallet > maxWallet {
					maxWallet = wallet
				}

				openedCnt = 0.0
				cnt++
			}
		}

		if openedCnt != 0 {
			h := highs[i]
			loss := 1 - (2*openedPrice-h)*openedCnt/maxWallet
			if loss > maxLoss {
				maxLoss = loss
				if maxLoss >= envMaxLoss {
					return TestMaxLoss
				}
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

	//speed = (wallet - StartDeposit) / float64(rnSum)
	//if speed > globalMaxSpeed*0.996 /* 1000.0*/ {
	//	saveStrategy += 2
	//	if speed > globalMaxSpeed {
	//		globalMaxSpeed = speed
	//	}
	//}

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
		} else if saveStrategy&1 == 1 {
			testData.StrategiesMaxWallet = append(testData.StrategiesMaxWallet, strategy)
		}

		fmt.Printf("\n %d %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			strategy.String(),
		)
	}

	return TestNoResult
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
	wallet, maxWallet := StartDeposit, StartDeposit
	var maxLoss, openedCnt, openedPrice, sl, tp float64
	var cnt, slCnt, saveStrategy int

	candles := candleData.Candles
	lows := candles[L]
	opens := candles[O]
	highs := candles[H]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] > float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt

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

	//speed = (wallet - StartDeposit) / float64(rnSum)
	//if speed > globalMaxSpeed*0.996 /* 1000.0*/ {
	//	saveStrategy += 2
	//	if speed > globalMaxSpeed {
	//		globalMaxSpeed = speed
	//	}
	//}

	safety := wallet / maxLoss
	if safety > globalMaxSafety*0.990 {
		saveStrategy += 4
		if safety > globalMaxSafety {
			globalMaxSafety = safety
		}
	}

	if saveStrategy > 0 {
		fmt.Printf("\n %d %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			strategy.String(),
		)

		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
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
	wallet, maxWallet := StartDeposit, StartDeposit
	var maxLoss, openedCnt, openedPrice, sl, tp float64
	var cnt, slCnt, saveStrategy int

	candles := candleData.Candles
	lows := candles[L]
	opens := candles[O]
	highs := candles[H]
	for i := 1; i < maxTimeIndex; i++ {
		o := opens[i]
		if openedCnt == 0 {
			if 10000*ind1[i-1]/ind2[i-1] > float64(10000+strategy.Op) {
				openedPrice = o
				openedCnt = wallet / openedPrice
				wallet -= openedPrice * openedCnt

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

	//speed = (wallet - StartDeposit) / float64(rnSum)
	//if speed > globalMaxSpeed*0.996 /* 1000.0*/ {
	//	saveStrategy += 2
	//	if speed > globalMaxSpeed {
	//		globalMaxSpeed = speed
	//	}
	//}

	safety := wallet / maxLoss
	if safety > globalMaxSafety*0.990 {
		saveStrategy += 4
		if safety > globalMaxSafety {
			globalMaxSafety = safety
		}
	}

	if saveStrategy > 0 {
		fmt.Printf("\n %d %s %s %s %s",
			saveStrategy,
			color.New(color.FgHiGreen).Sprintf("%5d%%", int(100*(wallet-StartDeposit)/StartDeposit)),
			color.New(color.FgHiRed).Sprintf("%4.1f%%", (maxLoss)*100.0),
			color.New(color.BgBlue).Sprintf("%4d", cnt),
			strategy.String(),
		)

		if saveStrategy&4 == 4 {
			testData.StrategiesMaxSafety = append(testData.StrategiesMaxSafety, strategy)
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
	TestNoResult testResultSign = iota + 1
	TestMaxLoss
	TestNotEnoughCnt
	TestNoStopLosses
)

func prepareTestPairs(envTestPairs string) {
	pairs := strings.Split(envTestPairs, ";")
	for _, pair := range pairs {
		candleData := getCandleData(pair)
		apiHandler.downloadPairCandles(candleData)

		favoriteStrategies := getTestData(pair)
		if !favoriteStrategies.restore() {
			candleData.testPair()
			tgBot.sendTestFile(favoriteStrategies.getFileName())
		} else {
			showStrategies(favoriteStrategies.StrategiesMaxSafety)
			showStrategies(favoriteStrategies.StrategiesMaxWallet)
		}
	}
}

func showStrategies(strategies []Strategy) {
	for _, strategy := range strategies {
		strategy.show()
	}
}

func (strategy Strategy) show() {
	globalMaxWallet, globalMaxSafety, envMinCnt = 0, 0, 0
	candleData := strategy.getCandleData()
	apiHandler.downloadPairCandles(candleData)
	ind1 := candleData.getIndicatorValue(strategy.Ind1)
	ind2 := candleData.getIndicatorValue(strategy.Ind2)
	strategy.Type.getTestStrategyFunction(candleData)(strategy, ind1, ind2, strategy.getTestData(), candleData.index())
}

func prepareTestStrategies(envTestStrategies string) {
	params := strings.Split(envTestStrategies, "}{")
	params[0] = params[0][1:]
	params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]

	for _, param := range params {
		getStrategy(param).show()
	}
}
