package main

import (
	"fmt"
	"net/http"
	"github.com/navikt/webhookproxy/webhook"
	"github.com/navikt/webhookproxy/events"
	"time"
	"bytes"
	"github.com/navikt/webhookproxy/context"
	"encoding/json"
)

type webhookRequest struct {
	request *http.Request
	body    []byte
	eventType string
	webhook *webhook.Webhook
}

var proxyClient = &http.Client{
	Timeout: time.Second * 5,
}

func (s *server) handlePingEvent(w http.ResponseWriter, r *http.Request) error {
	var pingEvent events.PingEvent
	if err := json.Unmarshal(context.RequestBodyFromContext(r.Context()), &pingEvent); err != nil {
		return err
	}

	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("content-type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(struct {
		Zen string `json:"zen"`
	}{
		Zen: pingEvent.Zen,
	})

	return nil
}

func (s *server) forwardRequest(w http.ResponseWriter, r *http.Request) error {
	wh := context.WebhookFromContext(r.Context())

	fmt.Printf("Forwarding request to %v\n", wh.Url)
	res, err := proxyClient.Post(wh.Url, "application/json", bytes.NewReader(context.RequestBodyFromContext(r.Context())))

	if err != nil {
		return appError{http.StatusInternalServerError, err.Error()}
	}

	fmt.Printf("%v\n", res)

	return nil
}

func (s *server) proxyHook(w http.ResponseWriter, r *http.Request) error {
	// ping events are not proxied
	if r.Header.Get("X-Github-Event") == "ping" {
		return s.handlePingEvent(w, r)
	}

	return s.forwardRequest(w, r)
}

func (s *server) newWebhook(w http.ResponseWriter, r *http.Request) error {
	var webhookRequest webhook.CreateWebhookRequest
	if err := json.Unmarshal(context.RequestBodyFromContext(r.Context()), &webhookRequest); err != nil {
		return err
	}

	wh, err := webhook.New(webhookRequest)
	if err != nil {
		return err
	}

	wh.ProxyUrl = "http://localhost:8080/hook/" + wh.Id

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("content-type", "application/json")

	encoder := json.NewEncoder(w)
	encoder.Encode(wh)

	return nil
}

func (s *server) isAlive(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte("is alive"))
	return nil
}

func (s *server) isReady(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte("is ready"))

	return nil
}
