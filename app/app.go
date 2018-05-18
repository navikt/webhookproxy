package app

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/navikt/webhookproxy/middlewares"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"fmt"
	"os"
	"github.com/navikt/webhookproxy/errors"
)

type server struct {
	router *mux.Router
}

func NewServer() *server {
	return &server{mux.NewRouter()}
}

func (s *server) Initialize() {
	s.router.Use(middlewares.LogHandler, middlewares.ReadRequestBodyHandler)

	s.router.Methods(http.MethodGet).Path("/metrics").
		Handler(promhttp.Handler())
	s.router.Methods(http.MethodGet).Path("/isAlive").
		Handler(appHandlerFunc(s.isAlive))
	s.router.Methods(http.MethodGet).Path("/isReady").
		Handler(appHandlerFunc(s.isReady))

	s.router.Methods(http.MethodGet).Path("/hooks").
		Handler(appHandlerFunc(s.listWebhooks))
	s.router.Methods(http.MethodPost).Path("/hooks").
		Handler(appHandlerFunc(s.newWebhook))

	hookRouter := s.router.PathPrefix("/hooks").Subrouter()
	hookRouter.Use(middlewares.MustHaveWebhook)

	hookRouter.Methods(http.MethodPost).Path("/{id}").
		Headers("X-Github-Event", "ping").
		Handler(middlewares.MustHaveValidSignature(appHandlerFunc(s.handlePingEvent)))

	hookRouter.Methods(http.MethodPost).Path("/{id}").
		Handler(middlewares.MustHaveValidSignature(appHandlerFunc(s.proxyHook)))

	hookRouter.Methods(http.MethodGet).Path("/{id}").
		Handler(appHandlerFunc(s.listWebhook)).
		Name("webhook")

	hookRouter.Methods(http.MethodDelete).Path("/{id}").
		Handler(appHandlerFunc(s.deleteWebhook))
}

func (s *server) Run(listenAddr string) {
	httpServer := &http.Server{Addr: listenAddr, Handler: s.router}
	panic(httpServer.ListenAndServe())
}

type appHandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (originalFn appHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := originalFn(w, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		errors.RespondWithError(w, err)
	}
}
