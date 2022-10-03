package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

var apiHandler ApiInterface

var resolution string

const StartDeposit = float64(100000.0)

const Commission = float64(0.98)

var exchange string

func init() {
	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())

	exchange = os.Getenv("exchange")

	switch exchange {
	case "exmo":
		apiHandler = exmo
	case "bybit":
		apiHandler = bybit.init()
	case "binance":
		apiHandler = binance.init()
	default:
		log.Fatal("NO HANDLER")
	}
	resolution = os.Getenv("resolution")
	envMinCnt = toUint(os.Getenv("min_cnt"))
	envMaxLoss = s2f(os.Getenv("max_loss"))
	years = toInt(os.Getenv("years"))
	months = toInt(os.Getenv("months"))
	days = toInt(os.Getenv("days"))
}

func main() {
	//c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt, os.Kill)
	//go func() {
	//	for sig := range c {
	//		log.Printf("Stopped %+v", sig)
	//		pprof.StopCPUProfile()
	//		os.Exit(1)
	//	}
	//}()

	envTestPair := os.Getenv("testPair")
	envTestPairs := os.Getenv("testPairs")
	envTestStrategies := os.Getenv("testStrategies")

	CandleStorage = make(map[string]CandleData)
	FavoriteStrategyStorage = make(map[string]FavoriteStrategies)

	if envTestPair != "" {
		candleData := getCandleData(envTestPair)
		apiHandler.downloadPairCandles(candleData)
		candleData.testPair()
	}

	if envTestPairs != "" {
		prepareTestPairs(envTestPairs)
	}

	if envTestStrategies != "" {
		prepareTestStrategies(envTestStrategies)
	}

	fmt.Println("\nFINISHED")

}

func prepareTestPairs(envTestPairs string) {
	var sliceTestData []*FavoriteStrategies
	pairs := strings.Split(envTestPairs, ";")
	for _, pair := range pairs {
		candleData := getCandleData(pair)
		apiHandler.downloadPairCandles(candleData)

		testData := getTestData(pair)
		if !testData.restore() {
			candleData.testPair()
			testData.restore()
		}
		testData.saveToStorage()
		sliceTestData = append(sliceTestData, testData)
	}
	fillStrategyTestTimes(sliceTestData)
	HeapPermutation(sliceTestData, len(sliceTestData))
	testMatrixStrategies(strategiesTestMatrix)
}

func prepareTestStrategies(envTestStrategies string) {
	params := strings.Split(envTestStrategies, "}{")
	params[0] = params[0][1:]
	params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]

	var sliceTestData []*FavoriteStrategies
	var strategies []Strategy
	for _, param := range params {
		strategy := getStrategy(param)
		strategies = append(strategies, strategy)
		testData := getTestData(strategy.Pair)
		testData.StrategiesMaxWallet = []Strategy{strategy}
		sliceTestData = append(sliceTestData, testData)
	}
	for _, strategy := range getUniqueStrategies(strategies) {
		candleData := strategy.getCandleData()
		apiHandler.downloadPairCandles(candleData)
	}
	fillStrategyTestTimes(sliceTestData)
	HeapPermutation(sliceTestData, len(sliceTestData))
	testMatrixStrategies(strategiesTestMatrix)
}

func fillStrategyTestTimes(sliceFavoriteStrategies []*FavoriteStrategies) {
	for _, testData := range sliceFavoriteStrategies {
		testData.TotalStrategies = append(testData.StrategiesMaxSpeed, testData.StrategiesMaxWallet...)
		testData.TotalStrategies = append(testData.TotalStrategies, testData.StrategiesMaxSafety...)
		testData.CandleData = getCandleData(testData.Pair)
	}

	candleDataSlice := make([]*CandleData, len(sliceFavoriteStrategies), len(sliceFavoriteStrategies))

	for i, testData := range sliceFavoriteStrategies {
		candleDataSlice[i] = getCandleData(testData.Pair)
	}

	totalTimeMap := make(map[time.Time]bool)
	strategyTestTimes.exist = make(map[string]map[time.Time]bool)
	for _, candleData := range candleDataSlice {
		strategyTestTimes.exist[candleData.Pair] = timeSlicesToMap(candleData.Time)
		totalTimeMap = mergeTimeMaps(totalTimeMap, strategyTestTimes.exist[candleData.Pair])
	}

	totalTimeSlices := timeMapToSlices(totalTimeMap)
	sort.Slice(totalTimeSlices, func(i, j int) bool {
		return totalTimeSlices[i].Before(totalTimeSlices[j])
	})
	strategyTestTimes.totalTimes = totalTimeSlices

	for t, _ := range totalTimeMap {
		totalTimeMap[t] = false
	}
	for data, m := range strategyTestTimes.exist {
		strategyTestTimes.exist[data] = mergeTimeMaps(totalTimeMap, m)
	}

	strategyTestTimes.indexes = make(map[string]map[time.Time]int)
	for _, candleData := range candleDataSlice {
		strategyTestTimes.indexes[candleData.Pair] = make(map[time.Time]int)
		j := -1
		for _, t := range strategyTestTimes.totalTimes {
			if strategyTestTimes.exist[candleData.Pair][t] {
				j++
				strategyTestTimes.indexes[candleData.Pair][t] = j
			} else {
				strategyTestTimes.indexes[candleData.Pair][t] = -1
			}
		}
	}
}

func HeapPermutation(a []*FavoriteStrategies, size int) {
	if size == 1 {
		strategiesTestMatrix = append(strategiesTestMatrix, append(make([]*FavoriteStrategies, 0, len(a)), a...))
	}

	for i := 0; i < size; i++ {
		r := size - 1
		HeapPermutation(a, r)
		a[(size%2)*i], a[r] = a[r], a[(size%2)*i]
	}
}

func timeSlicesToMap(timeSlices ...[]time.Time) map[time.Time]bool {
	uniqueMap := map[time.Time]bool{}

	for _, timeSlice := range timeSlices {
		for _, v := range timeSlice {
			uniqueMap[v] = true
		}
	}

	return uniqueMap
}

func mergeTimeMaps(m1, m2 map[time.Time]bool) map[time.Time]bool {
	tmp := make(map[time.Time]bool)
	for t, b := range m1 {
		tmp[t] = b
	}
	for t, b := range m2 {
		tmp[t] = b
	}
	return tmp
}

func timeMapToSlices(uniqueMap map[time.Time]bool) []time.Time {
	result := make([]time.Time, 0, len(uniqueMap))

	for key := range uniqueMap {
		result = append(result, key)
	}

	return result
}
