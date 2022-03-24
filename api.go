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

func listenCandle(figi string) {
	err := streamClient.SubscribeCandle(figi, tf.CandleInterval1Hour, requestID())
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
	data := Storage[c.FIGI][tf.CandleInterval1Hour]
	Storage[c.FIGI][tf.CandleInterval1Hour] = *data.upsertCandle(c)
	fmt.Println("asd")
}

func downloadCandlesByFigi(figi string) {
	Storage[figi] = make(map[tf.CandleInterval]CandleData)
	data := &CandleData{}

	data.Indicators = make(map[IndicatorType]map[int]map[string][]float64)
	data.Indicators[IndicatorTypeSma] = make(map[int]map[string][]float64)
	data.Indicators[IndicatorTypeEma] = make(map[int]map[string][]float64)
	data.Indicators[IndicatorTypeDema] = make(map[int]map[string][]float64)
	//data.Indicators[IndicatorTypeAma] = make(map[float64]map[string][]float64)
	data.Indicators[IndicatorTypeTema] = make(map[int]map[string][]float64)

	data.Candles = make(map[string][]float64)

	now := time.Now().AddDate(0, 0, -1)
	start := now.AddDate(0, 0, -15)
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

func downloadCandles(tm time.Time, figi string, data *CandleData) {
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
		data.upsertCandle(&candle)
	}
	Storage[figi][tf.CandleInterval1Hour] = *data
	fmt.Printf("Кол-во свечей: %d\n", len(Storage[figi][tf.CandleInterval1Hour].Time))

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
