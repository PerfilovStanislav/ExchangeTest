package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"flag"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"time"
)

const StartDeposit = float64(100000.0)
const Comission = float64(0.06)

func main() {
	//_ = godotenv.Load()
	//ConnectDb()
	//defer Db.Close()

	figi := flag.String("figi", "BBG000B9XRY4", "example: BBG000B9XRY4")
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		for sig := range c {
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			pprof.StopCPUProfile()
			os.Exit(1)
		}
	}()

	Storage = make(map[string]map[tf.CandleInterval]CandleData)
	//restoreStorage()

	rand.Seed(time.Now().UnixNano()) // инициируем Seed рандома для функции requestID
	registerClient()
	//registerStreamClient()
	//listenCandle(*figi)
	downloadCandlesByFigi(*figi)
	fillIndicators(*figi)
	//test(*figi)

	//backupStorage()

	select {}
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

func restoreStorage() {
	dataIn := Decompress(ReadFromFile("storage.dat"))
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&Storage)
}

func backupStorage() {
	dataOut := Compress(EncodeToBytes(Storage))
	_ = ioutil.WriteFile("storage.dat", dataOut, 0644)
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

func test(figi string) {
	storage := Storage[figi][tf.CandleInterval1Hour]

	var maxSpeed = 0.0
	var maxWallet = 0.0

	var wg sync.WaitGroup
	for op := 1.0; op < 1.0050; op += 0.0005 {
		wg.Add(1)
		go testOp(&wg, &maxSpeed, &maxWallet, op, &storage)
	}
	wg.Wait()
}

func testOp(wg *sync.WaitGroup, maxSpeed *float64, maxWallet *float64, op float64, storage *CandleData) {
	defer wg.Done()

	var cl, wallet, openedPrice, speed float64
	show := false

	for _, iType1 := range BarTypes {
		for cl = 1.0; cl < 1.1; cl += 0.00125 {
			for _, iType2 := range BarTypes {
				for _, indicatorType1 := range IndicatorTypes {
					indicators1 := storage.Indicators[indicatorType1]
					for coef1, bars1 := range indicators1 {

						for _, indicatorType2 := range IndicatorTypes {
							indicators2 := storage.Indicators[indicatorType2]
							for coef2, bars2 := range indicators2 {

								wallet = StartDeposit
								rnOpen := 0
								rnSum := 0
								openedCnt := 0
								cnt := 0

								for i, _ := range storage.Time {
									if i == 0 {
										continue
									}

									if openedCnt == 0 {
										if bars1[iType1][i-1]/bars2[iType2][i-1] >= op {
											openedPrice = storage.Candles["O"][i]
											openedCnt = int(wallet / openedPrice)
											wallet -= (Comission + openedPrice) * float64(openedCnt)
											rnOpen = i
										}
									} else if storage.Candles["O"][i]/openedPrice >= cl {
										wallet += storage.Candles["O"][i] * float64(openedCnt)
										if wallet <= StartDeposit*0.85 {
											break
										}
										openedCnt = 0
										cnt++
										rnSum += i - rnOpen
									}
								}

								if openedCnt >= 1 {
									wallet += (openedPrice + Comission) * float64(openedCnt)
								}

								speed = (wallet - StartDeposit) / float64(rnSum)
								if cnt >= 10 && rnSum != 0.0 {
									if speed > /*(*maxSpeed)*0.9*/ 1000.0 {
										show = true
										if speed > *maxSpeed {
											*maxSpeed = speed
										}
									}
								}

								//if wallet-StartDeposit > *maxWallet {
								//	*maxWallet = wallet
								//	show = true
								//}

								if show {
									fmt.Printf("%s %s %s %s ⬆%s ⬇%s [%s %s %s] [%s %s %s] \n️️",
										color.New(color.FgHiGreen).Sprint(int(wallet)),
										color.New(color.BgBlue).Sprintf("%4d", cnt),
										color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
										color.New(color.FgHiRed).Sprintf("%7.2f", speed),
										color.New(color.BgHiGreen).Sprintf("%.4f", op),
										color.New(color.BgHiRed).Sprintf("%.5f", cl),

										color.New(color.FgHiBlue).Sprintf("%4s", indicatorType1),
										color.New(color.FgWhite).Sprint(iType1),
										color.New(color.FgHiWhite).Sprintf("%5.2f", coef1),

										color.New(color.FgHiBlue).Sprintf("%4s", indicatorType2),
										color.New(color.FgWhite).Sprint(iType2),
										color.New(color.FgHiWhite).Sprintf("%5.2f", coef2),
									)
								}

								show = false

							}
						}
					}
				}
			}
		}
	}
	fmt.Println("===\n")
}
