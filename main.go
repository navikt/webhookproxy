package main

import (
	"os"
	"github.com/navikt/webhookproxy/app"
)

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	server := app.NewServer()
	server.Initialize()
	server.Run(listenAddr)
}
