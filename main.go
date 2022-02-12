package main

import (
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"

	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
)

const StartDeposit = float64(10000.0)
const Comission = float64(0.06)

func main() {
	// BBG000B9XRY4 - apple
	_ = godotenv.Load()
	ConnectDb()
	defer Db.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		for sig := range c {
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			pprof.StopCPUProfile()
			os.Exit(1)
		}
	}()

	CandlesMain()
	testing()
}

func testing() {
	test()
	//debug(maxRn, 131, 130, 1.0, 1.0400009, 4, 3)
}

func test() {
	storage := CandleIndicatorStorage["BBG000B9XRY4"][tf.CandleInterval1Hour]

	var op, cl, wallet, maxWallet, openedPrice float64
	maxWallet = 0.0

	for op = 1; op < 1.015; op += 0.00125 { // 0.00005
		for cl = 1.0; cl < 1.1; cl += 0.00125 {
			for a := 1; a <= 4; a++ {
				for b := 1; b <= 4; b++ {
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
											if getIndicator(bars1, a)[i-1]/getIndicator(bars2, b)[i-1] >= op {
												openedPrice = storage.Candles.O[i]
												openedCnt = int(wallet / openedPrice) // - 1
												wallet -= (Comission + openedPrice) * float64(openedCnt)
												rnOpen = i
											}
										} else if storage.Candles.O[i]/openedPrice >= cl {
											wallet += storage.Candles.O[i] * float64(openedCnt)
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

									if wallet > maxWallet {
										maxWallet = wallet
										log.Println(int(maxWallet), op, cl, a, b, indicatorType1, coef1, indicatorType2, coef2, cnt, rnSum)
									}

								}
							}
						}
					}
				}
			}
		}
	}
}

func getIndicator(bars Bars, x int) []float64 {
	switch x {
	case 1:
		return bars.C
	case 2:
		return bars.O
	case 3:
		return bars.H
	case 4:
		return bars.L
	}
	return nil
}
