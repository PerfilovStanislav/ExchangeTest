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

//var client *tf.SandboxRestClient
//var streamClient *tf.StreamingClient
//var logger *log.Logger

type Tinkoff struct {
	ApiClient    *tf.SandboxRestClient
	StreamClient *tf.StreamingClient
	Account      *tf.Account
}

func (tinkoff *Tinkoff) getApiClient() *tf.SandboxRestClient {
	if tinkoff.ApiClient == nil {
		tinkoff.ApiClient = tf.NewSandboxRestClient(os.Getenv("token"))
	}
	return tinkoff.ApiClient
}

func (tinkoff *Tinkoff) register(token string) {
	//tinkoff.ApiClient = tf.NewSandboxRestClient(token)
	//tinkoff.Account = tinkoff.registerAccount()
	//tinkoff.Clear()
	//tinkoff.StreamClient = tinkoff.registerStreamClient(token)
	//tinkoff.setBalance(tf.USD, 100000)
	//tinkoff.initCandleListener()
}

func (tinkoff *Tinkoff) registerAccount() *tf.Account {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	log.Println("Регистрация обычного счета в песочнице")
	account, err := tinkoff.getApiClient().Register(ctx, tf.AccountTinkoff)
	if err != nil {
		log.Fatalln(errorHandle(err))
	}
	log.Printf("%+v\n", account)
	return &account
}

func (tinkoff *Tinkoff) Clear() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := tinkoff.getApiClient().Clear(ctx, tinkoff.Account.ID)
	if err != nil {
		log.Fatalln(err)
	}
}

func (tinkoff *Tinkoff) registerStreamClient(token string) *tf.StreamingClient {
	logger := log.New(os.Stdout, "*", log.LstdFlags)

	streamClient, err := tf.NewStreamingClient(logger, token)
	if err != nil {
		log.Fatalln(err)
	}
	return streamClient
}

func (tinkoff *Tinkoff) initCandleListener() {
	go func() {
		err := tinkoff.StreamClient.RunReadLoop(func(event interface{}) error {
			fmt.Printf("-> %+v\n", event)
			switch sdkEvent := event.(type) {
			case tf.CandleEvent:
				newCandleEvent(tinkoff, sdkEvent.Candle)
				return nil
			default:
				fmt.Printf("sdkEvent %+v", sdkEvent)
			}

			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}
	}()
}

func (tinkoff *Tinkoff) setBalance(currency tf.Currency, balance float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := tinkoff.getApiClient().SetCurrencyBalance(ctx, tinkoff.Account.ID, currency, balance)
	if err != nil {
		log.Fatalln(err)
	}
}

func (tinkoff *Tinkoff) downloadCandlesByFigi(data *CandleData) {
	data.Candles = make(map[BarType][]float64)

	endDate := time.Now().AddDate(0, 0, 7)
	startDate := endDate.AddDate(-1, 0, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	for startDate.Before(endDate) {
		from := startDate
		to := startDate.AddDate(0, 0, 7)
		candles, err := tinkoff.getApiClient().Candles(ctx, from, to, data.Interval, data.Figi)
		if err != nil {
			fmt.Sprintln(err)
			log.Fatalln(err)
		}

		if len(candles) == 0 {
			break
		}

		for _, candle := range candles {
			data.upsertCandle(candle)
		}
		//fmt.Println("Sleep")
		//time.Sleep(time.Minute * time.Duration(2))
		fmt.Printf("Кол-во свечей: %d\n", data.len())

		startDate = to
	}
	data.save()
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
