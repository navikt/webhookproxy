package webhook

import (
	"fmt"
	"time"
	"crypto/sha1"
	"encoding/hex"
)

type CreateWebhookRequest struct {
	Name   string `json:"name"`
	Team   string `json:"team"`
	Url    string `json:"url"`
	Secret []byte `json:"secret"`
}

type Webhook struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Team     string `json:"team"`
	Url      string `json:"url"`
	Secret   []byte `json:"secret"`
	ProxyUrl string `json:"proxy_url"`
	CreatedAt time.Time `json:"created_at"`
}

var webhooks = map[string]*Webhook{}

func Lookup(team string, name string) *Webhook {
	return Get(getId(team, name))
}

func getId(team string, name string) string {
	idHash := sha1.New()
	idHash.Write([]byte(team))
	idHash.Write([]byte(name))
	return hex.EncodeToString(idHash.Sum(nil))
}

func New(request CreateWebhookRequest) (*Webhook, error) {
	id := getId(request.Team, request.Name)

	if Get(id) != nil {
		return nil, fmt.Errorf("webhook already exists")
	}

	webhook := &Webhook{
		Id: id,
		Name: request.Name,
		Team: request.Team,
		Url: request.Url,
		Secret: request.Secret,
		CreatedAt: time.Now(),
	}

	return Save(webhook)
}

func Save(webhook *Webhook) (*Webhook, error) {
	webhooks[webhook.Id] = webhook
	return webhook, nil
}

func Get(id string) *Webhook {
	if hook, ok := webhooks[id]; !ok {
		return nil;
	} else {
		return hook
	}
}

func Delete(id string) error {
	delete(webhooks, id)
	return nil
}
