package webhook

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("Create new webhook", func(t *testing.T) {
		got, err := New(CreateWebhookRequest{
			Name: "my-awesome-hook",
			Team: "cool-team-name",
			Url: "http://internal-server.tld/hook",
			Secret: []byte("foobar"),
		})
		want := &Webhook{
			Name: "my-awesome-hook",
			Team: "cool-team-name",
			Url: "http://internal-server.tld/hook",
			Secret: []byte("foobar"),
		}

		if err != nil {
			t.Errorf("New() error = %v", err)
			return
		}

		want.Id = got.Id
		want.CreatedAt = got.CreatedAt

		if !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})

	t.Run("Create duplicate should fail", func(t *testing.T) {
		req := CreateWebhookRequest{
			Name: "my-awesome-hook",
			Team: "cool-team-name",
			Url: "http://internal-server.tld/hook",
			Secret: []byte("foobar"),
		}

		New(req)
		_, err := New(req)

		if err == nil {
			t.Errorf("New() error = duplicate should fail")
			return
		}
	})
}
