package main

import (
	"fmt"
	"github.com/joho/godotenv"
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

func init() {
	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())

	switch os.Getenv("exchange") {
	case "exmo":
		apiHandler = exmo
	default:
		apiHandler = exmo
	}
	resolution = os.Getenv("resolution")
	envMinCnt = toInt(os.Getenv("min_cnt"))
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
	TestStorage = make(map[string]TestData)

	if envTestPair != "" {
		candleData := getCandleData(envTestPair)
		apiHandler.downloadPairCandles(candleData)
		candleData.parallelTestPair()
	}

	if envTestPairs != "" {
		testPairs(envTestPairs)
	}

	if envTestStrategies != "" {
		testStrategies(envTestStrategies)
	}

	fmt.Println("!")

	////tinkoff.Open("BBG000B9XRY4", 2)
	////tinkoff.Close("BBG000B9XRY4", 2)
	////ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	////defer cancel()
	////p, _ := tinkoff.ApiClient.Portfolio(ctx, tinkoff.Account.ID)
	////fmt.Printf("%+v", p)
	//
	////listenCandles(tinkoff)
	//
	//select {}

}

func testPairs(envTestPairs string) {
	var sliceTestData []*TestData
	envTestOperationPairs := strings.Split(envTestPairs, ";")
	for _, pair := range envTestOperationPairs {
		candleData := getCandleData(pair)
		apiHandler.downloadPairCandles(candleData)

		testData := getTestData(pair)
		if !testData.restore() {
			candleData.parallelTestPair()
			testData.restore()
		}
		testData.saveToStorage()
		sliceTestData = append(sliceTestData, testData)
	}
	fillOperationTestTimes(sliceTestData)
	HeapPermutation(sliceTestData, len(sliceTestData))
	testMatrixOperations(operationsTestMatrix)
}

func testStrategies(envTestStrategies string) {
	params := strings.Split(envTestStrategies, "}{")
	params[0] = params[0][1:]
	params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]

	var sliceTestData []*TestData
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
	fillOperationTestTimes(sliceTestData)
	HeapPermutation(sliceTestData, len(sliceTestData))
	testMatrixOperations(operationsTestMatrix)
}

func fillOperationTestTimes(operationsForTest []*TestData) {
	for _, testData := range operationsForTest {
		testData.TotalStrategies = append(testData.StrategiesMaxSpeed, testData.StrategiesMaxWallet...)
		testData.TotalStrategies = append(testData.TotalStrategies, testData.StrategiesMaxSafety...)
		testData.CandleData = getCandleData(testData.Pair)
	}

	candleDataSlice := make([]*CandleData, len(operationsForTest), len(operationsForTest))

	for i, testData := range operationsForTest {
		candleDataSlice[i] = getCandleData(testData.Pair)
	}

	totalTimeMap := make(map[time.Time]bool)
	operationTestTimes.exist = make(map[string]map[time.Time]bool)
	for _, candleData := range candleDataSlice {
		operationTestTimes.exist[candleData.Pair] = timeSlicesToMap(candleData.Time)
		totalTimeMap = mergeTimeMaps(totalTimeMap, operationTestTimes.exist[candleData.Pair])
	}

	totalTimeSlices := timeMapToSlices(totalTimeMap)
	sort.Slice(totalTimeSlices, func(i, j int) bool {
		return totalTimeSlices[i].Before(totalTimeSlices[j])
	})
	operationTestTimes.totalTimes = totalTimeSlices

	for t, _ := range totalTimeMap {
		totalTimeMap[t] = false
	}
	for data, m := range operationTestTimes.exist {
		operationTestTimes.exist[data] = mergeTimeMaps(totalTimeMap, m)
	}

	operationTestTimes.indexes = make(map[string]map[time.Time]int)
	for _, candleData := range candleDataSlice {
		operationTestTimes.indexes[candleData.Pair] = make(map[time.Time]int)
		j := -1
		for _, t := range operationTestTimes.totalTimes {
			if operationTestTimes.exist[candleData.Pair][t] {
				j++
				operationTestTimes.indexes[candleData.Pair][t] = j
			} else {
				operationTestTimes.indexes[candleData.Pair][t] = -1
			}
		}
	}
}

func HeapPermutation(a []*TestData, size int) {
	if size == 1 {
		operationsTestMatrix = append(operationsTestMatrix, append(make([]*TestData, 0, len(a)), a...))
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
