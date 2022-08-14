package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var exmo Exmo

type ApiParams map[string]string

var years, months, days int

func (exmo Exmo) downloadPairCandles(candleData *CandleData) {
	if candleData.restore() {
		return
	}

	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(years, months, days).Unix()

	for startDate < endDate {
		from := startDate
		to := startDate + 125*3600

		candleHistory := exmo.apiGetCandles(candleData.Pair, resolution, from, to)

		startDate = to

		if candleHistory.isEmpty() {
			continue
		}
		for _, c := range candleHistory.Candles {
			candleData.upsertCandle(c.transform())
		}
		fmt.Printf("%s - %s +%d\n",
			time.Unix(from, 0).Format("02.01.06 15"),
			time.Unix(to, 0).Format("02.01.06 15"),
			len(candleHistory.Candles),
		)
		time.Sleep(time.Millisecond * time.Duration(50))
	}
	fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	candleData.fillIndicators()
	candleData.save()
	candleData.backup()
}

func (exmo Exmo) apiGetCandles(symbol, resolution string, from, to int64) ExmoCandleHistoryResponse {
	params := ApiParams{
		"symbol":     symbol,
		"resolution": resolution,
		"from":       i2s(from),
		"to":         i2s(to),
	}

	bts, err := exmo.apiQuery("candles_history", params)

	var candleHistory ExmoCandleHistoryResponse
	err = json.Unmarshal(bts, &candleHistory)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return candleHistory
}

func (exmo Exmo) apiQuery(method string, params ApiParams) ([]byte, error) {
	postParams := url.Values{}
	postParams.Add("nonce", nonce())
	if params != nil {
		for k, value := range params {
			postParams.Add(k, value)
		}
	}
	postContent := postParams.Encode()

	sign := doSign(postContent, exmo.Secret)

	req, _ := http.NewRequest("POST", "https://api.exmo.com/v1.1/"+method, bytes.NewBuffer([]byte(postContent)))
	req.Header.Set("Key", exmo.Key)
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
