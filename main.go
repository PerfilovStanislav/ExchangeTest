package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"
)

const StartDeposit = float64(100000.0)
const Commission = float64(0.06)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		for sig := range c {
			log.Printf("Stopped %v", sig)
			pprof.StopCPUProfile()
			os.Exit(1)
		}
	}()

	Storage = make(map[string]map[tf.CandleInterval]CandleData)

	rand.Seed(time.Now().UnixNano())
	registerClient()
	registerStreamClient()
	//restoreStorage()

	testHandler()
	listenCandle("BBG000B9XRY4", tf.CandleInterval1Hour)

	//backupStorage()

	select {}
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

func restoreStorage() {
	dataIn := Decompress(ReadFromFile("storage.dat"))
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&Storage)
}

func backupStorage() {
	dataOut := Compress(EncodeToBytes(Storage))
	_ = ioutil.WriteFile("storage.dat", dataOut, 0644)
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
