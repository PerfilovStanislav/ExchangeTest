package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
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
		apiHandler.downloadCandlesByFigi(candleData)
		candleData.testFigi()
	}

	var operationsForTest []*TestData
	if envTestOperations != "" {
		envTestOperationParams := strings.Split(envTestOperations, ";")
		for _, param := range envTestOperationParams {
			candleData := getCandleData(param + ".hour")
			apiHandler = getApiHandler(envTestFigiInterval)
			if !candleData.restore() {
				apiHandler.downloadCandlesByFigi(candleData)
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

func getFigiAndInterval(str string) (string, tf.CandleInterval) {
	param := strings.Split(str, ".")
	return param[0], tf.CandleInterval(param[1])
}

func EncodeToBytes(p interface{}) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uncompressed size (bytes): ", len(buf.Bytes()))
	return buf.Bytes()
}

func Compress(s []byte) []byte {

	zipbuf := bytes.Buffer{}
	zipped := gzip.NewWriter(&zipbuf)
	zipped.Write(s)
	zipped.Close()
	fmt.Println("compressed size (bytes): ", len(zipbuf.Bytes()))
	return zipbuf.Bytes()
}

func Decompress(s []byte) []byte {
	rdr, _ := gzip.NewReader(bytes.NewReader(s))
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		log.Fatal(err)
	}
	rdr.Close()
	fmt.Println("uncompressed size (bytes): ", len(data))
	return data
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ReadFromFile(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

// parallel processes the data in separate goroutines.
func parallel(start, stop int, fn func(<-chan int)) {
	count := stop - start
	if count < 1 {
		return
	}

	procs := runtime.GOMAXPROCS(0)
	if procs > count {
		procs = count
	}

	c := make(chan int, count)
	for i := start; i < stop; i++ {
		c <- i
	}
	close(c)

	var wg sync.WaitGroup
	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(c)
		}()
	}
	wg.Wait()
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

func figiInterval(figi string, interval tf.CandleInterval) string {
	return fmt.Sprintf("%s_%s", figi, interval)
}
