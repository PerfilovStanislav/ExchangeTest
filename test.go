package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	//_ "github.com/lib/pq"
)

type TestData struct {
	FigiInterval string
	//Figi                string
	//Interval            tf.CandleInterval
	MaxWalletOperations []OperationParameter
	MaxSpeedOperations  []OperationParameter
	TotalOperations     []OperationParameter
	CandleData          *CandleData
}

var TestStorage map[string]TestData

func initTestData(figiInterval string) *TestData {
	data := TestStorage[figiInterval]
	data.FigiInterval = figiInterval
	return &data
}

func getTestData(figiInterval string) *TestData {
	data, ok := TestStorage[figiInterval]
	if ok == false {
		return initTestData(figiInterval)
	}
	return &data
}

func (testData *TestData) restore() bool {
	fileName := fmt.Sprintf("tests_%s.dat", testData.FigiInterval)
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&testData)

	return true
}

func (testData *TestData) backup() {
	testData.MaxWalletOperations = testData.MaxWalletOperations[maxInt(len(testData.MaxWalletOperations)-35, 0):]
	testData.MaxSpeedOperations = testData.MaxSpeedOperations[maxInt(len(testData.MaxSpeedOperations)-115, 0):]
	dataOut := EncodeToBytes(testData)
	_ = ioutil.WriteFile(fmt.Sprintf("tests_%s.dat", testData.FigiInterval), dataOut, 0644)
}

func (testData *TestData) saveToStorage() {
	TestStorage[testData.FigiInterval] = *testData
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func (candleData *CandleData) testFigi() {
	testData := getTestData(candleData.FigiInterval)

	var globalMaxSpeed = 0.0
	var globalMaxWallet = 0.0

	parallel(0, 12, func(ys <-chan int) {
		for y := range ys {
			testFigi(&globalMaxSpeed, &globalMaxWallet, y*5, candleData, testData)
		}
	})

	testData.backup()
}

func testFigi(globalMaxSpeed *float64, globalMaxWallet *float64, op int, candleData *CandleData, testData *TestData) {
	var wallet, openedPrice, speed, maxWallet, maxLoss float64
	var saveOperation int

	for _, barType1 := range BarTypes {
		for cl := 0; cl < 750; cl += 25 {
			for _, barType2 := range BarTypes {
				for _, indicatorType1 := range IndicatorTypes {
					indicators1 := candleData.Indicators[indicatorType1]
					for coef1, bars1 := range indicators1 {

						for _, indicatorType2 := range IndicatorTypes {
							indicators2 := candleData.Indicators[indicatorType2]
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

								for i, _ := range candleData.Time {
									if i == 0 {
										continue
									}

									if openedCnt == 0 {
										if bars1[barType1][i-1]*10000/bars2[barType2][i-1] >= float64(10000+op) {
											openedPrice = candleData.Candles["O"][i]
											openedCnt = int(wallet / (Commission + openedPrice))
											wallet -= (Commission + openedPrice) * float64(openedCnt)
											rnOpen = i
										}
									} else {
										o := candleData.Candles["O"][i]
										if o*10000/openedPrice >= float64(10000+cl) {
											wallet += o * float64(openedCnt)

											if wallet > maxWallet {
												maxWallet = wallet
											}

											l := candleData.Candles["L"][i]
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
										candleData.FigiInterval,
										op,
										IndicatorParameter{indicatorType1, barType1, coef1},
										cl,
										IndicatorParameter{indicatorType2, barType2, coef2},
									}
									if saveOperation&1 == 1 {
										testData.MaxWalletOperations = append(testData.MaxWalletOperations, operation)
									}
									if saveOperation&2 == 2 {
										testData.MaxSpeedOperations = append(testData.MaxSpeedOperations, operation)
									}

									fmt.Printf("\n %s %s %s %s ⬆%s ⬇%s [%s %s %s] [%s %s %s]️️ %s",
										color.New(color.FgHiGreen).Sprintf("%6d", int(wallet-StartDeposit)),
										color.New(color.BgBlue).Sprintf("%4d", cnt),
										color.New(color.FgHiYellow).Sprintf("%5d", rnSum),
										color.New(color.FgHiRed).Sprintf("%7.2f", speed),
										color.New(color.BgHiGreen).Sprintf("%3d", op),
										color.New(color.BgHiRed).Sprintf("%3d", cl),

										color.New(color.FgHiBlue).Sprintf("%2d", indicatorType1),
										color.New(color.FgWhite).Sprint(barType1),
										color.New(color.FgHiWhite).Sprintf("%2d", coef1),

										color.New(color.FgHiBlue).Sprintf("%2d", indicatorType2),
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

func (candleData *CandleData) getIndicatorValue(indicator IndicatorParameter) []float64 {
	return candleData.Indicators[indicator.IndicatorType][indicator.Coef][indicator.BarType]
}

func (candleData *CandleData) getIndicatorRatio(operation OperationParameter, index int) float64 {
	return candleData.getIndicatorValue(operation.Ind1)[index] / candleData.getIndicatorValue(operation.Ind2)[index]
}
