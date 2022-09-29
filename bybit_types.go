package main

import (
	"github.com/frankrap/bybit-api/rest"
	"time"
)

type Bybit struct {
	Key    string
	Secret string
	//AvailableDeposit float64
	//Balance          CurrencyBalanceResponse
	//OpenedOrder      OpenedOrder
	//StopLossOrderId  int64
}

func (bybit Bybit) transform(c rest.OHLCLinear) Candle {
	return Candle{
		c.Low,
		c.Open,
		c.Close,
		c.High,
		(c.Low + c.Open) * 0.5,
		(c.Low + c.Close) * 0.5,
		(c.Low + c.High) * 0.5,
		(c.Open + c.Close) * 0.5,
		(c.Open + c.High) * 0.5,
		(c.Close + c.High) * 0.5,
		(c.Low + c.Open + c.Close) / 3.0,
		(c.Low + c.Open + c.High) / 3.0,
		(c.Low + c.Close + c.High) / 3.0,
		(c.Open + c.Close + c.High) / 3.0,
		time.Unix(c.OpenTime, 0),
	}
}

//type OpenedOrder struct {
//	Strategy
//	OpenedPrice float64
//}
//
//type Currency string
//
//func (candleHistory ExmoCandleHistoryResponse) isEmpty() bool {
//	return len(candleHistory.Candles) == 0
//}
//
//func (order OpenedOrder) isEmpty() bool {
//	return order.Strategy.Pair == ""
//}
//
//type Price float64
//
//type CurrencyBalanceResponse struct {
//	USDT Price `json:"USDT,string"`
//	ETC  Price `json:"ETC,string"`
//	UNI  Price `json:"UNI,string"`
//	ALGO Price `json:"ALGO,string"`
//}
//
//func (c ExmoCandle) transform() Candle {
//	return Candle{
//		c.L,
//		c.O,
//		c.C,
//		c.H,
//		(c.L + c.O) * 0.5,
//		(c.L + c.C) * 0.5,
//		(c.L + c.H) * 0.5,
//		(c.O + c.C) * 0.5,
//		(c.O + c.H) * 0.5,
//		(c.C + c.H) * 0.5,
//		(c.L + c.O + c.C) / 3.0,
//		(c.L + c.O + c.H) / 3.0,
//		(c.L + c.C + c.H) / 3.0,
//		(c.O + c.C + c.H) / 3.0,
//		time.Unix(c.T/1000, 0),
//	}
//}
//
//type ExmoCandleHistoryResponse struct {
//	S       string       `json:"s"`
//	Candles []ExmoCandle `json:"candles"`
//}
//
//type ExmoCandle struct {
//	T int64   `json:"t"`
//	O float64 `json:"o"`
//	C float64 `json:"c"`
//	H float64 `json:"h"`
//	L float64 `json:"l"`
//}
//
//type OrderResponse struct {
//	Result   bool   `json:"result"`
//	Error    string `json:"error"`
//	OrderID  int    `json:"order_id"`
//	ClientID int    `json:"client_id"`
//}
//
//func (response OrderResponse) isSuccess() bool {
//	return response.Error == ""
//}
//
//type StopOrderResponse struct {
//	ClientID         int    `json:"client_id"`
//	ParentOrderID    int64  `json:"parent_order_id"`
//	ParentOrderIDStr string `json:"parent_order_id_str"`
//}
//
//func (response StopOrderResponse) isSuccess() bool {
//	return response.ParentOrderID > 0
//}
//
//type UserInfoResponse struct {
//	//UID        int                     `json:"uid"`
//	//ServerDate int                     `json:"server_date"`
//	Balances CurrencyBalanceResponse `json:"balances"`
//	//Reserved   CurrencyBalanceResponse `json:"reserved"`
//}
