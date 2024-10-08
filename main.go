package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"
)

var apiHandler ApiInterface

var resolution string

var threads int

const StartDeposit = float64(1000000.0)

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
	envLastMonthCnt = toInt(os.Getenv("min_last_month_cnt"))
	envMinCnt = toInt(os.Getenv("min_cnt"))
	envMaxLoss = s2f(os.Getenv("max_loss"))
	years = toInt(os.Getenv("years"))
	months = toInt(os.Getenv("months"))
	days = toInt(os.Getenv("days"))

	{
		params := strings.Split(os.Getenv("tp"), ";")
		tpMin = int(s2i(params[0]))
		tpMax = int(s2i(params[1]))
		tpDif = int(s2i(params[2]))
	}

	{
		params := strings.Split(os.Getenv("open"), ";")
		opMin = int(s2i(params[0]))
		opMax = int(s2i(params[1]))
		opDif = int(s2i(params[2]))
	}

	threads = toInt(os.Getenv("threads"))
	runtime.GOMAXPROCS(threads)

	//tgBot.init()
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

	envTestPairs := os.Getenv("test_pairs")
	envTestStrategies := os.Getenv("test_strategies")

	CandleStorage = make(map[string]CandleData)
	FavoriteStrategyStorage = make(map[string]FavoriteStrategies)

	if envTestPairs != "" {
		prepareTestPairs(envTestPairs)
	}

	if envTestStrategies != "" {
		prepareTestStrategies(envTestStrategies)
	}

	fmt.Println("\nFINISHED")
}

//func timeSlicesToMap(timeSlices ...[]time.Time) map[time.Time]bool {
//	uniqueMap := map[time.Time]bool{}
//
//	for _, timeSlice := range timeSlices {
//		for _, v := range timeSlice {
//			uniqueMap[v] = true
//		}
//	}
//
//	return uniqueMap
//}
//
//func mergeTimeMaps(m1, m2 map[time.Time]bool) map[time.Time]bool {
//	tmp := make(map[time.Time]bool)
//	for t, b := range m1 {
//		tmp[t] = b
//	}
//	for t, b := range m2 {
//		tmp[t] = b
//	}
//	return tmp
//}
//
//func timeMapToSlices(uniqueMap map[time.Time]bool) []time.Time {
//	result := make([]time.Time, 0, len(uniqueMap))
//
//	for key := range uniqueMap {
//		result = append(result, key)
//	}
//
//	return result
//}
