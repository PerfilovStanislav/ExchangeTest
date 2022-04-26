package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"sync"
	//_ "github.com/lib/pq"
)

var tests Tests

type Tests struct {
	MaxWalletOperations []OperationParameter
	MaxSpeedOperations  []OperationParameter
}

func (tests Tests) backup(figi string, interval tf.CandleInterval) {
	tests.MaxWalletOperations = tests.MaxWalletOperations[maxInt(len(tests.MaxWalletOperations)-3, 0):]
	tests.MaxSpeedOperations = tests.MaxSpeedOperations[maxInt(len(tests.MaxSpeedOperations)-100, 0):]
	dataOut := EncodeToBytes(tests)
	_ = ioutil.WriteFile(fmt.Sprintf("tests_%s_%s.dat", figi, interval), dataOut, 0644)
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func (tests Tests) restore(figi string, interval tf.CandleInterval) {
	dataIn := ReadFromFile(fmt.Sprintf("tests_%s_%s.dat", figi, interval))
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&tests)
}

func (data *CandleData) testFigi() {
	var globalMaxSpeed = 0.0
	var globalMaxWallet = 0.0

	var wg sync.WaitGroup
	for op := 0; op < 60; op += 5 {
		wg.Add(1)
		go testOp(&wg, &globalMaxSpeed, &globalMaxWallet, op, data)
	}
	wg.Wait()
}

func testOp(wg *sync.WaitGroup, globalMaxSpeed *float64, globalMaxWallet *float64, op int, data *CandleData) {
	defer wg.Done()

	var wallet, openedPrice, speed, maxWallet, maxLoss float64
	var saveOperation int

	for _, barType1 := range BarTypes {
		for cl := 0; cl < 750; cl += 25 {
			for _, barType2 := range BarTypes {
				for _, indicatorType1 := range IndicatorTypes {
					indicators1 := data.Indicators[indicatorType1]
					for coef1, bars1 := range indicators1 {

						for _, indicatorType2 := range IndicatorTypes {
							indicators2 := data.Indicators[indicatorType2]
						out:
							for coef2, bars2 := range indicators2 {

								saveOperation = 0
								wallet = StartDeposit
								maxWallet = StartDeposit
								maxLoss = 0
								rnOpen := 0
								rnSum := 0
								openedCnt := 0
								cnt := 0

								for i, _ := range data.Time {
									if i == 0 {
										continue
									}

									if openedCnt == 0 {
										if bars1[barType1][i-1]*10000/bars2[barType2][i-1] >= float64(10000+op) {
											openedPrice = data.Candles["O"][i]
											openedCnt = int(wallet / (Commission + openedPrice))
											wallet -= (Commission + openedPrice) * float64(openedCnt)
											rnOpen = i
										}
									} else {
										o := data.Candles["O"][i]
										if o*10000/openedPrice >= float64(10000+cl) {
											wallet += o * float64(openedCnt)

											if wallet > maxWallet {
												maxWallet = wallet
											}

											l := data.Candles["L"][i]
											loss := 1 - l*float64(openedCnt)/maxWallet
											if loss > maxLoss {
												maxLoss = loss
												if maxLoss >= 0.05 {
													continue out
												}
											}

											openedCnt = 0
											cnt++
											rnSum += i - rnOpen
										}
									}

								}

								if openedCnt >= 1 {
									wallet += (openedPrice + Commission) * float64(openedCnt)
								}

								if wallet > *globalMaxWallet {
									*globalMaxWallet = wallet
									saveOperation += 1
								}

								speed = (wallet - StartDeposit) / float64(rnSum)
								if cnt >= 25 && rnSum > 1 {
									if speed > (*globalMaxSpeed)*0.995 /* 1000.0*/ {
										saveOperation += 1
										if speed > *globalMaxSpeed {
											*globalMaxSpeed = speed
										}
									}
								}

								if saveOperation > 0 {
									operation := OperationParameter{
										op, cl,
										IndicatorParameter{indicatorType1, barType1, coef1},
										IndicatorParameter{indicatorType2, barType2, coef2},
										data.Figi, data.Interval,
									}
									if saveOperation&1 == 1 {
										tests.MaxWalletOperations = append(tests.MaxWalletOperations, operation)
									}
									if saveOperation&2 == 2 {
										tests.MaxSpeedOperations = append(tests.MaxSpeedOperations, operation)
									}

									fmt.Printf("\n %s %s %s %s ⬆%s ⬇%s [%s %s %s] [%s %s %s]️️ %s",
										color.New(color.FgHiGreen).Sprintf("%6d", int(wallet-StartDeposit)),
										color.New(color.BgBlue).Sprintf("%4d", cnt),
										color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
										color.New(color.FgHiRed).Sprintf("%7.2f", speed),
										color.New(color.BgHiGreen).Sprintf("%3d", op),
										color.New(color.BgHiRed).Sprintf("%3d", cl),

										color.New(color.FgHiBlue).Sprintf("%5s", indicatorType1),
										color.New(color.FgWhite).Sprint(barType1),
										color.New(color.FgHiWhite).Sprintf("%2d", coef1),

										color.New(color.FgHiBlue).Sprintf("%5s", indicatorType2),
										color.New(color.FgWhite).Sprint(barType2),
										color.New(color.FgHiWhite).Sprintf("%2d", coef2),
										color.New(color.FgHiRed).Sprintf("%4.2f%%", (maxLoss)*100.0),
									)
								}

							}
						}
					}
				}
			}
		}
	}
}

func testOperations(data *CandleData) {
	var wallet, openedPrice, speed, maxWallet, maxSpeed float64
	show := false

	length := len(tests.MaxWalletOperations)
	for x := 0; x < length; x++ {
		for y := 0; y < length; y++ {
			for z := 0; z < length; z++ {
				operation1 := tests.MaxWalletOperations[x]
				operation2 := tests.MaxWalletOperations[y]
				operation3 := tests.MaxWalletOperations[z]

				wallet = StartDeposit
				rnOpen := 0
				rnSum := 0
				openedCnt := 0
				cnt := 0

				var cl = 0
				for i, _ := range data.Time {
					if i == 0 {
						continue
					}

					if openedCnt == 0 {
						if 10000*data.getIndicatorRatio(operation1, i-1) >= float64(10000+operation1.Op) {
							cl = operation1.Cl
						} else if 10000*data.getIndicatorRatio(operation2, i-1) >= float64(10000+operation2.Op) {
							cl = operation2.Cl
						} else if 10000*data.getIndicatorRatio(operation3, i-1) >= float64(10000+operation3.Op) {
							cl = operation3.Cl
						}
						if cl > 0 {
							openedPrice = data.Candles["O"][i]
							openedCnt = int(wallet / (Commission + openedPrice))
							wallet -= (Commission + openedPrice) * float64(openedCnt)
							rnOpen = i
						}
					} else {
						o := data.Candles["O"][i]
						if 10000*o/openedPrice >= float64(10000+cl) {
							wallet += o * float64(openedCnt)

							cl = 0
							openedCnt = 0
							cnt++
							rnSum += i - rnOpen
						}
					}

				}

				if openedCnt >= 1 {
					wallet += (openedPrice + Commission) * float64(openedCnt)
				}

				speed = (wallet - StartDeposit) / float64(rnSum)

				if speed > maxSpeed {
					show = true
					maxSpeed = speed
				}

				if wallet > maxWallet {
					maxWallet = wallet
					show = true
				}

				if show {
					//fmt.Printf("\n %s %s %s %s ⬆%s ⬇%s [%s %s %s] [%s %s %s]️️ %s",
					fmt.Printf("\n %s %s %s %s %s %s %s",
						color.New(color.FgHiGreen).Sprintf("%7d", int(wallet-StartDeposit)),
						color.New(color.BgBlue).Sprintf("%4d", cnt),
						color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
						color.New(color.FgHiRed).Sprintf("%7.2f", speed),
						showOperation(operation1),
						showOperation(operation2),
						showOperation(operation3),
						//
						//color.New(color.FgHiBlue).Sprintf("%5s", indicatorType2),
						//color.New(color.FgWhite).Sprint(barType2),
						//color.New(color.FgHiWhite).Sprintf("%2d", coef2),
					)
				}

				show = false
			}
		}
	}
}

func showOperation(operation OperationParameter) string {
	return fmt.Sprintf("{%s %s}", showIndicator(operation.Ind1), showIndicator(operation.Ind2))
}

func showIndicator(indicator IndicatorParameter) string {
	return fmt.Sprintf("[%s %s %s]",
		color.New(color.FgHiBlue).Sprintf("%5s", indicator.IndicatorType),
		color.New(color.FgWhite).Sprint(indicator.BarType),
		color.New(color.FgHiWhite).Sprintf("%2d", indicator.Coef),
	)
}

func (data *CandleData) getIndicatorValue(indicator IndicatorParameter) []float64 {
	return data.Indicators[indicator.IndicatorType][indicator.Coef][indicator.BarType]
}

func (data *CandleData) getIndicatorRatio(operation OperationParameter, index int) float64 {
	return data.getIndicatorValue(operation.Ind1)[index] / data.getIndicatorValue(operation.Ind2)[index]
}
