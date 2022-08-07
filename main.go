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

const StartDeposit = float64(100000.0)

const Commission = float64(0.98)

var apiHandler ApiInterface

func main() {
	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())
	//c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt, os.Kill)
	//go func() {
	//	for sig := range c {
	//		log.Printf("Stopped %+v", sig)
	//		pprof.StopCPUProfile()
	//		os.Exit(1)
	//	}
	//}()

	//restore := flag.Bool("restore", os.Getenv("restore") == "true", "Restore")
	envTestFigiInterval := os.Getenv("testFigiInterval")
	envTestOperations := os.Getenv("testOperations")

	CandleStorage = make(map[string]CandleData)
	TestStorage = make(map[string]TestData)

	if envTestFigiInterval != "" {
		//figi, interval := getFigiAndInterval(envTestFigiInterval)
		candleData := getCandleData(envTestFigiInterval + ".hour")
		apiHandler = getApiHandler(envTestFigiInterval)
		apiHandler.downloadCandlesForSymbol(candleData)
		candleData.testFigi()
	}

	var operationsForTest []*TestData
	if envTestOperations != "" {
		envTestOperationParams := strings.Split(envTestOperations, ";")
		for _, param := range envTestOperationParams {
			candleData := getCandleData(param + ".hour")
			apiHandler = getApiHandler(envTestFigiInterval)
			if !candleData.restore() {
				apiHandler.downloadCandlesForSymbol(candleData)
			}

			testData := getTestData(param + ".hour")
			if !testData.restore() {
				candleData.testFigi()
				testData.restore()
			}
			testData.saveToStorage()
			operationsForTest = append(operationsForTest, testData)
		}
		fillOperationTestTimes(operationsForTest)
		HeapPermutation(operationsForTest, len(operationsForTest))
		testMatrixOperations(operationsTestMatrix)
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

func getApiHandler(figi string) ApiInterface {
	switch len(figi) {
	//case 12:
	//	return &tinkoff
	default:
		return &exmo
	}
}

func fillOperationTestTimes(operationsForTest []*TestData) {
	for _, testData := range operationsForTest {
		testData.TotalOperations = append(testData.MaxSpeedOperations, testData.MaxWalletOperations...)
		testData.CandleData = getCandleData(testData.FigiInterval)
	}

	candleDataSlice := make([]*CandleData, len(operationsForTest), len(operationsForTest))

	for i, testData := range operationsForTest {
		candleDataSlice[i] = getCandleData(testData.FigiInterval)
	}

	totalTimeMap := make(map[time.Time]bool)
	operationTestTimes.exist = make(map[string]map[time.Time]bool)
	for _, candleData := range candleDataSlice {
		operationTestTimes.exist[candleData.FigiInterval] = timeSlicesToMap(candleData.Time)
		totalTimeMap = mergeTimeMaps(totalTimeMap, operationTestTimes.exist[candleData.FigiInterval])
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
		operationTestTimes.indexes[candleData.FigiInterval] = make(map[time.Time]int)
		j := -1
		for _, t := range operationTestTimes.totalTimes {
			if operationTestTimes.exist[candleData.FigiInterval][t] {
				j++
				operationTestTimes.indexes[candleData.FigiInterval][t] = j
			} else {
				operationTestTimes.indexes[candleData.FigiInterval][t] = -1
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
