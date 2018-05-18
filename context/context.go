package context

import (
	"net/http"
	"io/ioutil"
	"context"
	"github.com/navikt/webhookproxy/webhook"
)

type requestContextKey int
const (
	requestBodyKey requestContextKey = iota
	webhookKey
)


func NewContextWithRequestBody(ctx context.Context, r *http.Request) (context.Context, error) {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, requestBodyKey, body), nil
}

func RequestBodyFromContext(ctx context.Context) []byte {
	return ctx.Value(requestBodyKey).([]byte)
}

func NewContextWithWebhook(ctx context.Context,webhook *webhook.Webhook) context.Context {
	return context.WithValue(ctx, webhookKey, webhook)
}

func WebhookFromContext(ctx context.Context) *webhook.Webhook {
	return ctx.Value(webhookKey).(*webhook.Webhook)
}

