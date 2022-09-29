package main

import (
	"fmt"
	"github.com/frankrap/bybit-api/rest"
	"log"
	"os"
	"time"
)

var bybit Bybit

const bybitBaseUrl = "https://api.bybit.com/" // "https://api-testnet.bybit.com/"

func (bybit Bybit) init() Bybit {
	bybit.Key = os.Getenv("bybit.key")
	bybit.Secret = os.Getenv("bybit.secret")
	return bybit
}

func (bybit Bybit) downloadPairCandles(candleData *CandleData) {
	b := rest.New(nil, bybitBaseUrl, bybit.Key, bybit.Secret, false)

	if candleData.restore() {
		return
	}

	seconds := int64(toInt(resolution)) * 60
	endDate := (time.Now().Unix() / seconds) * seconds
	startDate := (time.Now().AddDate(years, months, days).Unix()/seconds)*seconds + 1

	for startDate < endDate {
		_, _, candles, err := b.LinearGetKLine("ETCUSDT", resolution, startDate, 200)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		fmt.Printf("%s +%d\n",
			time.Unix(startDate, 0).Format("02.01.06 15:04"),
			len(candles),
		)

		startDate += seconds * 200

		if len(candles) == 0 {
			continue
		}
		for _, c := range candles {
			candleData.upsertCandle(bybit.transform(c))
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
