package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func getStrategy(str string) Strategy {
	p := strings.Fields(str)

	ind := func(t, b, c string) Indicator {
		return Indicator{
			IndicatorType: IndicatorType(toInt(t)),
			BarType:       BarType(0).value(b),
			Coef:          toInt(c),
		}
	}

	return Strategy{
		Pair: p[0],
		Op:   toInt(p[1]),
		Ind1: ind(p[4], p[5], p[6]),
		Tp:   toInt(p[2]),
		Ind2: ind(p[8], p[9], p[10]),
	}
}

func toInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		fmt.Printf("%+v", err)
		return -100
	}
	return i
}

func toUint(str string) uint {
	return uint(toInt(str))
}

func EncodeToBytes(p interface{}) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uncompressed size (bytes): ", len(buf.Bytes()))
	return buf.Bytes()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ReadFromFile(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

// parallel processes the data in separate goroutines.
func parallel(start, stop int, fn func(<-chan int)) {
	count := stop - start
	if count < 1 {
		return
	}

	procs := runtime.GOMAXPROCS(0)
	if procs > count {
		procs = count
	}

	c := make(chan int, count)
	for i := start; i < stop; i++ {
		c <- i
	}
	close(c)

	var wg sync.WaitGroup
	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(c)
		}()
	}
	wg.Wait()
}

func f2s(x float64) string {
	return fmt.Sprintf("%v", x)
}

func s2f(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func i2s(i int64) string {
	return strconv.FormatInt(i, 10)
}

func s2i(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func getCurrencies(pair string) (Currency, Currency) {
	split := strings.Split(pair, "_")
	return Currency(split[0]), Currency(split[1])
}

func getLeftCurrency(pair string) Currency {
	currency, _ := getCurrencies(pair)
	return currency
}

func getRightCurrency(pair string) Currency {
	_, currency := getCurrencies(pair)
	return currency
}

func getUniqueStrategies(strategies []Strategy) []Strategy {
	var uniqueStrategies []Strategy
	var symbols []string
	for _, strategy := range strategies {
		pair := strategy.Pair
		if sliceIndex(symbols, pair) == -1 {
			symbols = append(symbols, pair)
			uniqueStrategies = append(uniqueStrategies, strategy)
		}
	}
	return uniqueStrategies
}

func sliceIndex[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}

func Compress(s []byte) []byte {

	zipbuf := bytes.Buffer{}
	zipped := gzip.NewWriter(&zipbuf)
	zipped.Write(s)
	zipped.Close()
	fmt.Println("compressed size (bytes): ", len(zipbuf.Bytes()))
	return zipbuf.Bytes()
}

func Decompress(s []byte) []byte {
	rdr, _ := gzip.NewReader(bytes.NewReader(s))
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		log.Fatal(err)
	}
	rdr.Close()
	fmt.Println("uncompressed size (bytes): ", len(data))
	return data
}
