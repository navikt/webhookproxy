package app

import (
	"fmt"
	"net/http"
	"github.com/navikt/webhookproxy/webhook"
	"github.com/navikt/webhookproxy/events"
	"time"
	"bytes"
	"github.com/navikt/webhookproxy/context"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"github.com/navikt/webhookproxy/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	webhookProxyRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "webhooks_proxy_requests", Help: "number of requests proxied per hook"}, []string{"hook"},
	)
	webhooksCounter = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "webhooks_count", Help: "number of webhooks"},
	)
)

func init() {
	prometheus.MustRegister(webhookProxyRequestCount)
	prometheus.MustRegister(webhooksCounter)
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

func (s *server) proxyHook(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	wh := context.WebhookFromContext(r.Context())

	webhookProxyRequestCount.With(prometheus.Labels{"hook": wh.Id}).Inc()

	fmt.Printf("Forwarding request to %v\n", wh.Url)
	res, err := proxyClient.Post(wh.Url, "application/json", bytes.NewReader(context.RequestBodyFromContext(r.Context())))

	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, err.Error())
	}

	fmt.Fprintf(w,"%s", body)

	return nil
}

func (s *server) urlForWebhook(w *webhook.Webhook) (*url.URL, error) {
	u, err := s.router.Get("webhook").URL("id", w.Id)

	if err != nil {
		return nil, err
	}

	w.ProxyUrl = u.String()
	return u, nil
}

func (s *server) listWebhooks(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "application/json")

	webhooks := webhook.List()
	for _, wh := range webhooks {
		if _, err := s.urlForWebhook(wh); err != nil {
			return errors.NewAppError(http.StatusInternalServerError, err.Error())
		}
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(webhooks)

	return nil
}

func (s *server) listWebhook(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "application/json")

	wh := context.WebhookFromContext(r.Context())
	if _, err := s.urlForWebhook(wh); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, err.Error())
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(wh)

	return nil
}

func (s *server) deleteWebhook(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusNoContent)

	wh := context.WebhookFromContext(r.Context())
	webhook.Delete(wh.Id)

	return nil
}

func (s *server) newWebhook(w http.ResponseWriter, r *http.Request) error {
	var webhookRequest webhook.CreateWebhookRequest
	if err := json.Unmarshal(context.RequestBodyFromContext(r.Context()), &webhookRequest); err != nil {
		return errors.NewAppError(http.StatusBadRequest, "invalid secret, must be base64: " + err.Error())
	}

	wh, err := webhook.New(webhookRequest)
	if err != nil {
		return err
	}

	webhooksCounter.Inc()

	if _, err := s.urlForWebhook(wh); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, err.Error())
	}

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
