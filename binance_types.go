package main

import (
	binanceApi "github.com/adshao/go-binance/v2/futures"
	"time"
)

type Binance struct {
	Client *binanceApi.Client
}

func (binance Binance) transform(c *binanceApi.Kline) Candle {
	return Candle{
		s2f(c.Low),
		s2f(c.Open),
		s2f(c.Close),
		s2f(c.High),
		(s2f(c.Low) + s2f(c.Open)) * 0.5,
		(s2f(c.Low) + s2f(c.Close)) * 0.5,
		(s2f(c.Low) + s2f(c.High)) * 0.5,
		(s2f(c.Open) + s2f(c.Close)) * 0.5,
		(s2f(c.Open) + s2f(c.High)) * 0.5,
		(s2f(c.Close) + s2f(c.High)) * 0.5,
		(s2f(c.Low) + s2f(c.Open) + s2f(c.Close)) / 3.0,
		(s2f(c.Low) + s2f(c.Open) + s2f(c.High)) / 3.0,
		(s2f(c.Low) + s2f(c.Close) + s2f(c.High)) / 3.0,
		(s2f(c.Open) + s2f(c.Close) + s2f(c.High)) / 3.0,
		time.Unix(c.OpenTime/1000, 0),
	}
}
