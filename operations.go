package main

import (
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	//_ "github.com/lib/pq"
)

type OperationParameter struct {
	Open, Close float64
	Ind1, Ind2  IndicatorParameter
}

type IndicatorParameter struct {
	IndicatorType IndicatorType
	BarType       BarType
	Coef          int
}

var listenCandles = map[string]map[tf.CandleInterval][]OperationParameter{
	"BBG000B9XRY4": {
		tf.CandleInterval1Hour: []OperationParameter{
			{1.003, 1.0025,
				IndicatorParameter{IndicatorTypeSma, Low, 3},
				IndicatorParameter{IndicatorTypeDema, Close, 24},
			},
		},
		//tf.CandleInterval4Hour: []OperationParameter{{}},
	},
}
