package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var exmo Exmo

type Exmo struct {
	ApiClient    *tf.SandboxRestClient
	StreamClient *tf.StreamingClient
	Account      *tf.Account
}

func (exmo *Exmo) downloadCandlesByFigi(candleData *CandleData) {
	candleData.Candles = make(map[BarType][]float64)

	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(-2, 0, 0).Unix()

	for startDate < endDate {
		from := startDate
		to := startDate + 125*3600

		figi, _ := getFigiAndInterval(candleData.FigiInterval)

		params := ApiParams{
			"symbol":     figi,
			"resolution": "60",
			"from":       strconv.FormatInt(from, 10),
			"to":         strconv.FormatInt(to, 10),
		}

		bts, err := exmo.apiQuery("candles_history", params)

		type CandleHistory struct {
			S       string `json:"s"`
			Candles []struct {
				T int64   `json:"t"`
				O float64 `json:"o"`
				C float64 `json:"c"`
				H float64 `json:"h"`
				L float64 `json:"l"`
			} `json:"candles"`
		}

		var candleHistory CandleHistory
		err = json.Unmarshal(bts, &candleHistory)

		if err != nil {
			fmt.Sprintln(err)
			log.Fatalln(err)
		}

		startDate = to

		if candleHistory.S != "" {
			continue
		}
		for _, c := range candleHistory.Candles {
			candleData.upsertCandle(Candle{
				c.O, c.C, c.H, c.L, time.Unix(c.T, 0),
			})
		}
		fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	}

	candleData.fillIndicators()
	candleData.save()
	candleData.backup()
}

type ApiParams map[string]string

func (exmo *Exmo) apiQuery(method string, params ApiParams) ([]byte, error) {
	key := "K-c57e3128c287732d6371bc7934710ee62fe79f22"
	secret := "S-df7f03267f160fb6bf761d76699014f7060ceccb"

	postParams := url.Values{}
	postParams.Add("nonce", nonce())
	if params != nil {
		for k, value := range params {
			postParams.Add(k, value)
		}
	}
	postContent := postParams.Encode()

	sign := doSign(postContent, secret)

	req, _ := http.NewRequest("POST", "https://api.exmo.com/v1.1/"+method, bytes.NewBuffer([]byte(postContent)))
	req.Header.Set("Key", key)
	req.Header.Set("Sign", sign)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(postContent)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		return nil, errors.New("http status: " + resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func nonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func doSign(message string, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
