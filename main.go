package main

import (
	"log"
	"os"
)

func main() {

	app, err := New(os.Getenv("DB_HOST"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	if err != nil {
		log.Fatal(err)
		return
	}

	app.Run(os.Getenv("PORT"))
}
