package main

import (
	"context"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"math/rand"
	"os"
	"time"
	//_ "github.com/lib/pq"
)

var token = "t.ZNcVav8ge3MFAbSb0Y2ccwd-a9bPBkxKaPf0Yr_wD3Fc5tUpj8LX6gAg4RLlAUcIZm-KFKnImNMpeQf-2CmlbA"
var client *tf.SandboxRestClient
var streamClient *tf.StreamingClient
var logger *log.Logger

func listenCandle(figi string, interval tf.CandleInterval) {
	err := streamClient.SubscribeCandle(figi, interval, requestID())
	if err != nil {
		log.Fatalln(err)
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
				newCandleEvent(sdkEvent.Candle)
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

func newCandleEvent(c tf.Candle) {
	data := getStorageData(c.FIGI, c.Interval)
	data.upsertCandle(c)
	data.save()
	fmt.Printf("%v+", c)
}

func downloadCandlesByFigi(data *CandleData) {
	data.Candles = make(map[BarType][]float64)

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

		downloadCandles(start, data)
		start = start.AddDate(0, 0, 7)
	}
}

func downloadCandles(tm time.Time, data *CandleData) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	candles, err := client.Candles(ctx, tm.AddDate(0, 0, -7), tm, data.Interval, data.Figi)
	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	if len(candles) == 0 {
		return
	}

	for _, candle := range candles {
		data.upsertCandle(candle)
	}
	data.save()
	fmt.Printf("Кол-во свечей: %d\n", data.len())

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
