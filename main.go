package main

import (
	"log"
	"os"
)

func main() {

	storage, err := NewPostgres(os.Getenv("DB_HOST"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	if err != nil {
		log.Fatalf("App not able to start: %v", err)
	}

	defer storage.con.Close()

	app, _ := NewApp(storage)
	app.Run(os.Getenv("PORT"))
}
