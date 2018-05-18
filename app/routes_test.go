package app

import (
	"net/http"
	"testing"
	"net/http/httptest"
	"github.com/navikt/webhookproxy/webhook"
	"fmt"
	"strings"
	"time"
	"math/rand"
)

type MockClient struct {
	*http.Client
}
func executeRequest(s *server, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func checkResponseBody(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected body <%v>. Got <%v>\n", expected, actual)
	}
}

func newRandomWebhook(url string) *webhook.Webhook {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	wh, _ := webhook.New(webhook.CreateWebhookRequest{
		fmt.Sprintf("my-cool-webhook-%d", r.Int()),
		"awesome-team",
		url,
		[]byte("foobar"),
	})
	return wh
}

func clearWebhooks() {
	for _, w := range webhook.List() {
		webhook.Delete(w.Id)
	}
}

func Test_server_handlePingEvent(t *testing.T) {
	t.Run("empty signature header should not route to handler", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "ping")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, "{\"message\":\"malformed signature header\"}\n", w.Body.String())
	})

	t.Run("request with invalid signature should fail", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "ping")
		r.Header.Set("X-Hub-Signature", "sha1=aaff00bb")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusForbidden, w.Code)
		checkResponseBody(t, "{\"message\":\"invalid signature\"}\n", w.Body.String())
	})

	t.Run("request with ok headers should route to handler", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "ping")
		r.Header.Set("X-Hub-Signature", "sha1=dfb90a8c012eb0b97e6ec0865226bccedd723502")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusAccepted, w.Code)
		checkResponseBody(t, "{\"zen\":\"Mind your words, they are important.\"}\n", w.Body.String())
	})
}

func Test_server_proxyHook(t *testing.T) {
	t.Run("empty signature header should not route to handler", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "push")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, "{\"message\":\"malformed signature header\"}\n", w.Body.String())
	})

	t.Run("request with invalid signature should fail", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "push")
		r.Header.Set("X-Hub-Signature", "sha1=aaff00bb")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusForbidden, w.Code)
		checkResponseBody(t, "{\"message\":\"invalid signature\"}\n", w.Body.String())
	})

	t.Run("request with ok headers should route to handler", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello, client\n")
		}))
		defer ts.Close()

		wh := newRandomWebhook(ts.URL)
		defer clearWebhooks()
		r, _ := http.NewRequest("POST", "/hooks/" + wh.Id, strings.NewReader(`{"zen": "Mind your words, they are important."}`))
		r.Header.Set("X-Github-Event", "push")
		r.Header.Set("X-Hub-Signature", "sha1=dfb90a8c012eb0b97e6ec0865226bccedd723502")
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusOK, w.Code)
		checkResponseBody(t, "Hello, client\n", w.Body.String())
	})
}

func Test_server_listWebhook(t *testing.T) {
	t.Run("server should respond with error when webhook does not exist", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		r, _ := http.NewRequest("GET", "/hooks/dfb90a8c012eb0b97e6ec0865226bccedd723502", strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusNotFound, w.Code)
		checkResponseBody(t, "{\"message\":\"webhook does not exist\"}\n", w.Body.String())
	})

	t.Run("server should respond with webhook", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("GET", "/hooks/" + wh.Id, strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusOK, w.Code)
		bb, _ := wh.CreatedAt.MarshalJSON()
		jsonTime := string(bb)
		checkResponseBody(t, "{\"id\":\"" + wh.Id + "\",\"name\":\"" + wh.Name + "\",\"team\":\"" + wh.Team + "\",\"url\":\"" + wh.Url + "\",\"proxy_url\":\"/hooks/" + wh.Id + "\",\"created_at\":" + jsonTime + "}\n", w.Body.String())
	})
}

func Test_server_listWebhooks(t *testing.T) {
	t.Run("server should respond with an empty result when there is no webhooks", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		r, _ := http.NewRequest("GET", "/hooks", strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusOK, w.Code)
		checkResponseBody(t, "[]\n", w.Body.String())
	})

	t.Run("server should respond with list of webhooks", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("GET", "/hooks", strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusOK, w.Code)
		bb, _ := wh.CreatedAt.MarshalJSON()
		jsonTime := string(bb)
		checkResponseBody(t, "[{\"id\":\"" + wh.Id + "\",\"name\":\"" + wh.Name + "\",\"team\":\"" + wh.Team + "\",\"url\":\"" + wh.Url + "\",\"proxy_url\":\"/hooks/" + wh.Id + "\",\"created_at\":" + jsonTime + "}]\n", w.Body.String())
	})
}

func Test_server_deleteWebhook(t *testing.T) {
	t.Run("server should respond with error when webhook does not exist", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		r, _ := http.NewRequest("DELETE", "/hooks/dfb90a8c012eb0b97e6ec0865226bccedd723502", strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusNotFound, w.Code)
		checkResponseBody(t, "{\"message\":\"webhook does not exist\"}\n", w.Body.String())
	})

	t.Run("server should respond with list of webhooks", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		wh := newRandomWebhook("http://forward.tld/my-hook")
		defer clearWebhooks()
		r, _ := http.NewRequest("DELETE", "/hooks/" + wh.Id, strings.NewReader(""))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusNoContent, w.Code)

		r, _ = http.NewRequest("GET", "/hooks", strings.NewReader(""))
		w = executeRequest(s, r)

		checkResponseCode(t, http.StatusOK, w.Code)
		checkResponseBody(t, "[]\n", w.Body.String())
	})
}

func Test_server_newWebhook(t *testing.T) {
	t.Run("server should fail if secret is not base64", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		r, _ := http.NewRequest("POST", "/hooks", strings.NewReader(`{
	"name": "awesome-webhook",
	"team": "my-team-name",
	"url": "http://forward.tld/my-webhook",
	"secret": "asdf&()!!!"
}`))
		w := executeRequest(s, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, `{"message":"invalid secret, must be base64: illegal base64 data at input byte 4"}
`, w.Body.String())
	})

	t.Run("server should respond with webhook", func(t *testing.T) {
		s := NewServer()
		s.Initialize()

		r, _ := http.NewRequest("POST", "/hooks", strings.NewReader(`{
	"name": "awesome-webhook",
	"team": "my-team-name",
	"url": "http://forward.tld/my-webhook",
	"secret": "Zm9vYmFy"
}`))
		w := executeRequest(s, r)

		wh := webhook.Lookup("my-team-name", "awesome-webhook")

		checkResponseCode(t, http.StatusCreated, w.Code)
		bb, _ := wh.CreatedAt.MarshalJSON()
		jsonTime := string(bb)
		checkResponseBody(t, "{\"id\":\"" + wh.Id + "\",\"name\":\"awesome-webhook\",\"team\":\"my-team-name\",\"url\":\"http://forward.tld/my-webhook\",\"proxy_url\":\"/hooks/" + wh.Id + "\",\"created_at\":" + jsonTime + "}\n", w.Body.String())
	})
}

func Test_server_isAlive(t *testing.T) {
	s := NewServer()
	s.Initialize()

	r, _ := http.NewRequest("GET", "/isAlive", strings.NewReader(""))
	w := executeRequest(s, r)

	checkResponseCode(t, http.StatusOK, w.Code)
	checkResponseBody(t, "is alive", w.Body.String())
}

func Test_server_isReady(t *testing.T) {
	s := NewServer()
	s.Initialize()

	r, _ := http.NewRequest("GET", "/isReady", strings.NewReader(""))
	w := executeRequest(s, r)

	checkResponseCode(t, http.StatusOK, w.Code)
	checkResponseBody(t, "is ready", w.Body.String())
}
