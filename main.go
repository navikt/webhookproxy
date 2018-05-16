package main

import (
	"net/http"
	"fmt"
	"os"
	"github.com/navikt/webhookproxy/middlewares"
)

type appError struct {
	status  int
	message string
}

func (a appError) Status() int {
	return a.status
}

func (a appError) Error() string {
	return a.message
}

type server struct {}

type appHandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (originalFn appHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := originalFn(w, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		switch err := err.(type) {
		case appError:
			http.Error(w, err.Error(), err.Status())
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func Handler(appHandler appHandlerFunc, middlewareHandlers ...middlewares.Middleware) http.Handler {
	h := http.Handler(appHandler)
	for _, middleware := range middlewareHandlers {
		h = middleware(h)
	}
	h = middlewares.ReadRequestBodyHandler(h)
	h = middlewares.LogHandler(h)
	return h
}

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	server := &server{}

	http.Handle("/isAlive", Handler(server.isAlive))
	http.Handle("/isReady", Handler(server.isReady))
	http.Handle("/new", Handler(server.newWebhook, middlewares.MustHaveMethod("POST")))
	http.Handle("/hook/", Handler(server.proxyHook, middlewares.MustHaveValidSignature,
		middlewares.MustHaveHeader("X-Github-Event"), middlewares.MustHaveMethod("POST")))

	panic(http.ListenAndServe(listenAddr, nil))
}
