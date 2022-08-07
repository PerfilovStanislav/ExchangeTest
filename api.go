package main

type ApiInterface interface {
	downloadCandlesForSymbol(candleData *CandleData)
}
