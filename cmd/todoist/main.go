package main

import (
	"log"

	"github.com/haccht/todoist"
)

func main() {
	app, err := todoist.NewApplication()
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
