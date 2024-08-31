package main

import (
	"context"
	"fmt"
	bb "github.com/wuhewuhe/bybit.go.api"
	"github.com/wuhewuhe/bybit.go.api/models"
	"log"
	"os"
	"strings"
	"time"
)

var bybit Bybit

func (bybit Bybit) init() Bybit {
	key := os.Getenv("bybit.key")
	secret := os.Getenv("bybit.secret")

	bybit.client = bb.NewBybitHttpClient(key, secret, bb.WithBaseURL(bb.MAINNET))

	return bybit
}

func (bybit Bybit) downloadPairCandles(candleData *CandleData) {
	const limit = 1000
	if candleData.restore() {
		return
	}

	seconds := int64(toInt(resolution)) * 60
	startDate := (time.Now().AddDate(-years, -months, -days).Unix()/seconds)*seconds + 1
	endDate := (time.Now().Unix() / seconds) * seconds
	symbol := strings.ReplaceAll(candleData.Pair, "_", "")

	service := bybit.client.NewMarketKlineService()
	service.Category(models.CategorySpot)
	service.Symbol(symbol)
	service.Interval("60")
	service.Limit(limit)

	for startDate < endDate {
		service.Start(uint64(startDate * 1000))
		candles, err := service.Do(context.Background())
		if err != nil {
			log.Printf("%v", err)
			return
		}

		fmt.Printf("%s +%d\n",
			time.Unix(startDate, 0).Format("02.01.06 15:04"),
			len(candles.List),
		)

		startDate += seconds * limit

		if len(candles.List) == 0 {
			continue
		}
		for i := len(candles.List) - 1; i >= 0; i-- {
			candleData.upsertCandle(bybit.transform(candles.List[i]))
		}
		time.Sleep(time.Millisecond * time.Duration(50))
	}
	fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	candleData.fillIndicators()
	candleData.save()
	candleData.backup()
}

//func (exmo Exmo) apiGetCandles(symbol, resolution string, from, to int64) ExmoCandleHistoryResponse {
//	params := ApiParams{
//		"symbol":     symbol,
//		"resolution": resolution,
//		"from":       i2s(from),
//		"to":         i2s(to),
//	}
//
//	bts, err := exmo.apiQuery("candles_history", params)
//
//	var candleHistory ExmoCandleHistoryResponse
//	err = json.Unmarshal(bts, &candleHistory)
//
//	if err != nil {
//		fmt.Sprintln(err)
//		log.Fatalln(err)
//	}
//
//	return candleHistory
//}
//
//func (exmo Exmo) apiQuery(method string, params ApiParams) ([]byte, error) {
//	postParams := url.Values{}
//	postParams.Add("nonce", nonce())
//	if params != nil {
//		for k, value := range params {
//			postParams.Add(k, value)
//		}
//	}
//	postContent := postParams.Encode()
//
//	sign := doSign(postContent, exmo.Secret)
//
//	req, _ := http.NewRequest("POST", "https://api.exmo.com/v1.1/"+method, bytes.NewBuffer([]byte(postContent)))
//	req.Header.Set("Key", exmo.Key)
//	req.Header.Set("Sign", sign)
//	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
//	req.Header.Add("Content-Length", strconv.Itoa(len(postContent)))
//
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		return nil, err
//	}
//	defer resp.Body.Close()
//
//	if resp.Status != "200 OK" {
//		return nil, errors.New("http status: " + resp.Status)
//	}
//
//	return ioutil.ReadAll(resp.Body)
//}
//
//func nonce() string {
//	return fmt.Sprintf("%d", time.Now().UnixNano())
//}
//
//func doSign(message string, secret string) string {
//	mac := hmac.New(sha512.New, []byte(secret))
//	mac.Write([]byte(message))
//	return fmt.Sprintf("%x", mac.Sum(nil))
//}
