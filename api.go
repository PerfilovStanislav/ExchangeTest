package main

type ApiInterface interface {
	downloadPairCandles(candleData *CandleData)
}
