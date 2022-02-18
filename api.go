package main

import (
	"context"
	"fmt"
	ps "github.com/PerfilovStanislav/go-raw-postgresql-builder"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
	//_ "github.com/lib/pq"
)

var token = "t.ZNcVav8ge3MFAbSb0Y2ccwd-a9bPBkxKaPf0Yr_wD3Fc5tUpj8LX6gAg4RLlAUcIZm-KFKnImNMpeQf-2CmlbA"
var client *tf.SandboxRestClient
var streamClient *tf.StreamingClient
var logger *log.Logger

type ListenCandleData struct {
	StockIntervalId int64
	Time            time.Time
}

var listenCandles = map[string]map[string]ListenCandleData{ // interval => figi => stock_interval_id
	"hour": {
		"BBG000B9XRY4": {1, time.Unix(0, 0)},
		"BBG000BND699": {2, time.Unix(0, 0)},
	},
}

func Download() {
	registerStreamClient()

	initListening()
	defer streamClient.Close()

	for {
		time.Sleep(10 * time.Second)
	}
	//downloadCandlesByFigi("BBG000BND699")
	//startListening()
}

func initListening() {
	sql := `SELECT max(time) FROM candles WHERE stock_interval_id = $StockIntervalId`
	var t time.Time

	for interval, candles := range listenCandles {
		for figi, data := range candles {
			err := Db.Get(&t,
				ps.Sql{sql, struct{ StockIntervalId int64 }{data.StockIntervalId}}.String(),
			)
			if err != nil {
				log.Panic(err, sql)
			}
			candle := listenCandles[interval][figi]
			candle.Time = t
			listenCandles[interval][figi] = candle

			err = streamClient.SubscribeCandle(figi, tf.CandleInterval(interval), requestID())
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func registerClient() {
	client = tf.NewSandboxRestClient(token)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	log.Println("Регистрация обычного счета в песочнице")
	account, err := client.Register(ctx, tf.AccountTinkoff)
	if err != nil {
		log.Fatalln(errorHandle(err))
	}
	log.Printf("%+v\n", account)
}

func registerStreamClient() {
	logger = log.New(os.Stdout, "*", log.LstdFlags)

	var err error
	streamClient, err = tf.NewStreamingClient(logger, token)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		err = streamClient.RunReadLoop(func(event interface{}) error {
			logger.Printf("Got event %+v", event)
			switch sdkEvent := event.(type) {
			case tf.CandleEvent:
				newCandleEvent(&sdkEvent.Candle)
				return nil
			default:
				logger.Printf("sdkEvent %+v", sdkEvent)
			}

			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}
	}()
}

func newCandleEvent(c *tf.Candle) {
	data := listenCandles[string(c.Interval)][c.FIGI]
	if data.Time == c.TS {
		return
	}
	sql := ps.Sql{"INSERT INTO candles(stock_interval_id, time, o, c, h, l, v) SELECT " +
		strconv.FormatInt(data.StockIntervalId, 10) +
		", $TS, $OpenPrice, $ClosePrice, $HighPrice, $LowPrice, $Volume", c,
	}
	_, err := Db.Exec(sql.String())
	if err != nil {
		log.Fatalln(err)
	}
}

func downloadCandlesByFigi(figi string) {
	CandleIndicatorStorage[figi] = make(map[tf.CandleInterval]CandleIndicatorData)
	data := &CandleIndicatorData{}

	data.Indicators = make(map[IndicatorType]map[float64]Bars)
	data.Indicators[IndicatorTypeSma] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeEma] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeDema] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeAma] = make(map[float64]Bars)
	data.Indicators[IndicatorTypeTema] = make(map[float64]Bars)

	now := time.Now().AddDate(0, 0, 7)
	start := now.AddDate(-3, 0, 0)
	var apiCallCounter = 0
	for start.Before(now) {
		apiCallCounter += 1
		if apiCallCounter > 490 {
			fmt.Println("Sleep")
			time.Sleep(time.Minute * time.Duration(2))
			apiCallCounter = 1
		}

		downloadCandles(start, figi, data)
		start = start.AddDate(0, 0, 7)
	}
}

func downloadCandles(tm time.Time, figi string, data *CandleIndicatorData) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	candles, err := client.Candles(ctx, tm.AddDate(0, 0, -7), tm, tf.CandleInterval1Hour, figi)
	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	if len(candles) == 0 {
		return
	}

	for _, candle := range candles {
		data.Time = append(data.Time, candle.TS)
		data.Candles.O = append(data.Candles.O, candle.OpenPrice)
		data.Candles.C = append(data.Candles.C, candle.ClosePrice)
		data.Candles.H = append(data.Candles.H, candle.HighPrice)
		data.Candles.L = append(data.Candles.L, candle.LowPrice)
	}
	CandleIndicatorStorage[figi][tf.CandleInterval1Hour] = *data
	fmt.Printf("Кол-во свечей: %d\n", len(CandleIndicatorStorage[figi][tf.CandleInterval1Hour].Time))

}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Генерируем уникальный ID для запроса
func requestID() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

func errorHandle(err error) error {
	if err == nil {
		return nil
	}

	if tradingErr, ok := err.(tf.TradingError); ok {
		if tradingErr.InvalidTokenSpace() {
			tradingErr.Hint = "Do you use sandbox token in production environment or vise verse?"
			return tradingErr
		}
	}

	return err
}
