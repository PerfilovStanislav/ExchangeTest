package main

import (
	"context"
	"fmt"
	binanceApi "github.com/adshao/go-binance/v2/futures"
	"os"
	"strings"
	"time"
)

var binance Binance

func (binance Binance) init() Binance {
	binance.Client = binanceApi.NewClient(os.Getenv("binance.key"), os.Getenv("binance.secret"))
	return binance
}

func (binance Binance) downloadPairCandles(candleData *CandleData) {
	if candleData.restore() {
		return
	}

	interval := map[string]string{
		"15":  "15m",
		"30":  "30m",
		"60":  "1h",
		"240": "4h",
	}[resolution]

	seconds := map[string]int64{
		"15":  15,
		"30":  30,
		"60":  60,
		"240": 60 * 4,
	}[resolution] * 60

	endDate := (time.Now().Unix() / seconds) * seconds
	startDate := (time.Now().AddDate(years, months, days).Unix() / seconds) * seconds

	for startDate < endDate {
		candles, err := binance.Client.NewKlinesService().
			Symbol(strings.ReplaceAll(candleData.Pair, "_", "")).
			StartTime(startDate * 1000).
			Interval(interval).
			Limit(1000).
			Do(context.Background())

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("%s +%d\n",
			time.Unix(startDate, 0).Format("02.01.06 15:04"),
			len(candles),
		)

		startDate += seconds * 1000

		if len(candles) == 0 {
			continue
		}
		for _, c := range candles {
			candleData.upsertCandle(binance.transform(c))
		}
		time.Sleep(time.Millisecond * time.Duration(50))
	}
	fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	candleData.fillIndicators()
	candleData.save()
	candleData.backup()
}
