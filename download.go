package main

import (
	"context"
	"fmt"
	ps "github.com/PerfilovStanislav/go-raw-postgresql-builder"
	sdk "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
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
var client *sdk.SandboxRestClient
var streamClient *sdk.StreamingClient
var apiCallCounter int32 = 0
var err error
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
	rand.Seed(time.Now().UnixNano()) // инициируем Seed рандома для функции requestID

	registerClient()
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
			err = Db.Get(&t,
				ps.Sql{sql, struct{ StockIntervalId int64 }{data.StockIntervalId}}.String(),
			)
			if err != nil {
				log.Panic(err, sql)
			}
			candle := listenCandles[interval][figi]
			candle.Time = t
			listenCandles[interval][figi] = candle

			err = streamClient.SubscribeCandle(figi, sdk.CandleInterval(interval), requestID())
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func registerClient() {
	client = sdk.NewSandboxRestClient(token)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	log.Println("Регистрация обычного счета в песочнице")
	account, err := client.Register(ctx, sdk.AccountTinkoff)
	if err != nil {
		log.Fatalln(errorHandle(err))
	}
	log.Printf("%+v\n", account)
}

func registerStreamClient() {
	logger = log.New(os.Stdout, "*", log.LstdFlags)

	streamClient, err = sdk.NewStreamingClient(logger, token)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		err = streamClient.RunReadLoop(func(event interface{}) error {
			logger.Printf("Got event %+v", event)
			switch sdkEvent := event.(type) {
			case sdk.CandleEvent:
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

func newCandleEvent(c *sdk.Candle) {
	data := listenCandles[string(c.Interval)][c.FIGI]
	if data.Time == c.TS {
		return
	}
	sql := ps.Sql{"INSERT INTO candles(stock_interval_id, time, o, c, h, l, v) SELECT " +
		strconv.FormatInt(data.StockIntervalId, 10) +
		", $TS, $OpenPrice, $ClosePrice, $HighPrice, $LowPrice, $Volume", c,
	}
	_, err = Db.Exec(sql.String())
	if err != nil {
		log.Fatalln(err)
	}
}

func downloadCandlesByFigi(figi string) {
	now := time.Now().AddDate(0, 0, 7)
	start := now.AddDate(-3, 0, 0)
	for start.Before(now) {

		apiCallCounter += 1
		if apiCallCounter > 490 {
			fmt.Println("Sleep")
			time.Sleep(time.Second * time.Duration(60*2))
			apiCallCounter = 1
		}

		downloadCandles(start, figi)
		start = start.AddDate(0, 0, 7)
	}
}

func downloadCandles(tm time.Time, figi string) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	candles, err := client.Candles(ctx, tm.AddDate(0, 0, -7), tm, sdk.CandleInterval1Hour, figi)
	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	if len(candles) == 0 {
		return
	}

	sql := ps.Sql{
		"INSERT INTO candles(stock_interval_id, time, o, c, h, l, v) VALUES $Values",
		struct{ Values ps.Sql }{
			ps.Sql{Query: "\n(2, $TS, $OpenPrice, $ClosePrice, $HighPrice, $LowPrice, $Volume)", Data: candles},
		},
	}
	result, err := Db.Exec(sql.String())
	if err != nil {
		fmt.Println(result, sql)
		panic(err)
	}
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

	if tradingErr, ok := err.(sdk.TradingError); ok {
		if tradingErr.InvalidTokenSpace() {
			tradingErr.Hint = "Do you use sandbox token in production environment or vise verse?"
			return tradingErr
		}
	}

	return err
}
