package main

import (
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"sync"
	//_ "github.com/lib/pq"
)

func testHandler() {
	data := initStorageData("BBG000B9XRY4", tf.CandleInterval1Hour)

	downloadCandlesByFigi(data)
	//fillIndicators(data)
	//test(*figi)
}

func test(data *CandleData) {
	var maxSpeed = 0.0
	var maxWallet = 0.0

	var wg sync.WaitGroup
	for op := 1.0; op < 1.0050; op += 0.0005 {
		wg.Add(1)
		go testOp(&wg, &maxSpeed, &maxWallet, op, data)
	}
	wg.Wait()
}

func testOp(wg *sync.WaitGroup, maxSpeed *float64, maxWallet *float64, op float64, data *CandleData) {
	defer wg.Done()

	var cl, wallet, openedPrice, speed float64
	show := false

	for _, barType1 := range BarTypes {
		for cl = 1.0; cl < 1.1; cl += 0.00125 {
			for _, barType2 := range BarTypes {
				for _, indicatorType1 := range IndicatorTypes {
					indicators1 := data.Indicators[indicatorType1]
					for coef1, bars1 := range indicators1 {

						for _, indicatorType2 := range IndicatorTypes {
							indicators2 := data.Indicators[indicatorType2]
							for coef2, bars2 := range indicators2 {

								wallet = StartDeposit
								rnOpen := 0
								rnSum := 0
								openedCnt := 0
								cnt := 0

								for i, _ := range data.Time {
									if i == 0 {
										continue
									}

									if openedCnt == 0 {
										if bars1[barType1][i-1]/bars2[barType2][i-1] >= op {
											openedPrice = data.Candles["O"][i]
											openedCnt = int(wallet / openedPrice)
											wallet -= (Commission + openedPrice) * float64(openedCnt)
											rnOpen = i
										}
									} else if data.Candles["O"][i]/openedPrice >= cl {
										wallet += data.Candles["O"][i] * float64(openedCnt)
										if wallet <= StartDeposit*0.85 {
											break
										}
										openedCnt = 0
										cnt++
										rnSum += i - rnOpen
									}
								}

								if openedCnt >= 1 {
									wallet += (openedPrice + Commission) * float64(openedCnt)
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
										color.New(color.FgWhite).Sprint(barType1),
										color.New(color.FgHiWhite).Sprintf("%5.2f", coef1),

										color.New(color.FgHiBlue).Sprintf("%4s", indicatorType2),
										color.New(color.FgWhite).Sprint(barType2),
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
