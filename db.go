package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"os"
)

var Db *sqlx.DB

func ConnectDb() {
	var err error
	f := os.Getenv("DB_HOST")
	fmt.Println(f)
	Db, err = sqlx.Connect("pgx", fmt.Sprintf("host=%s user=%s password=%s dbname=%s "+
		"sslmode=disable port=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT")))
	if err != nil {
		panic(err)
	}
}
