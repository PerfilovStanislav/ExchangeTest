package main

type ApiInterface interface {
	downloadCandlesByFigi(candleData *CandleData)
}
