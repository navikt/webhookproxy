package middlewares

import (
	"net/http"
	"reflect"
	"testing"
	"net/http/httptest"
	"github.com/navikt/webhookproxy/context"
	"encoding/hex"
	"github.com/navikt/webhookproxy/webhook"
	"strings"
	"github.com/gorilla/mux"
)

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

func TestMustHaveWebhook(t *testing.T) {
	t.Run("Non-existing webhook should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("TestMustHaveWebhook() should not call next handler in chain")
		})

		handler := MustHaveWebhook(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/asdf", nil)

		handler.ServeHTTP(w, r)

		checkResponseCode(t, http.StatusNotFound, w.Code)
		checkResponseBody(t, "{\"message\":\"webhook does not exist\"}\n", w.Body.String())
	})
	t.Run("Webhook should be put in context", func(t *testing.T) {
		wh, _ := webhook.New(webhook.CreateWebhookRequest{
			"my-cool-webhook1",
			"my-team-name",
			"http://url-to-server.tld/hook",
			[]byte("foobar"),
		})

		nextHandlerCalled := false

		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true

			whFromContext := context.WebhookFromContext(r.Context())

			if !reflect.DeepEqual(wh, whFromContext) {
				t.Errorf("MustHaveWebhook() should set webhook in request context to <%v>, was <%v>", wh, whFromContext)
				return
			}
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, nil)
		r = mux.SetURLVars(r, map[string]string{"id": wh.Id})

		handler := MustHaveWebhook(dummyHandler)
		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveWebhook() should call next handler in chain")
			return
		}
	})
}

func TestMustHaveValidSignature(t *testing.T) {
	t.Run("No header should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveValidSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)

		handler.ServeHTTP(w, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, "{\"message\":\"malformed signature header\"}\n", w.Body.String())
	})

	t.Run("Invalid header should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveValidSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("X-Hub-Signature", "foo")

		handler.ServeHTTP(w, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, "{\"message\":\"malformed signature header\"}\n", w.Body.String())
	})
	t.Run("Invalid algo should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveValidSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("X-Hub-Signature", "sha2=aebfc")

		handler.ServeHTTP(w, r)

		checkResponseCode(t, http.StatusBadRequest, w.Code)
		checkResponseBody(t, "{\"message\":\"malformed signature header, unknown algo\"}\n", w.Body.String())
	})

	t.Run("Invalid signature should fail", func(t *testing.T) {
		wh, _ := webhook.New(webhook.CreateWebhookRequest{
			"my-cool-webhook2",
			"my-team-name",
			"http://url-to-server.tld/hook",
			[]byte("foobar"),
		})

		givenSignature := "816421f91f8bb65da114aef4616abf77052cccfe"

		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveValidSignature() should not call next handler in chain")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, strings.NewReader("Hello, World! This message has been altered."))
		r = mux.SetURLVars(r, map[string]string{"id": wh.Id})
		r.Header.Set("X-Hub-Signature", "sha1=" + givenSignature)

		handler := ReadRequestBodyHandler(MustHaveWebhook(MustHaveValidSignature(dummyHandler)))
		handler.ServeHTTP(w, r)

		checkResponseCode(t, http.StatusForbidden, w.Code)
		checkResponseBody(t, "{\"message\":\"invalid signature\"}\n", w.Body.String())
	})

	t.Run("Valid signature should pass", func(t *testing.T) {
		wh, _ := webhook.New(webhook.CreateWebhookRequest{
			"my-cool-webhook3",
			"my-team-name",
			"http://url-to-server.tld/hook",
			[]byte("foobar"),
		})

		givenSignature := "816421f91f8bb65da114aef4616abf77052cccfe"
		nextHandlerCalled := false

		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
		})

		handler := ReadRequestBodyHandler(MustHaveWebhook(MustHaveValidSignature(dummyHandler)))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, strings.NewReader("Hello, World!"))
		r = mux.SetURLVars(r, map[string]string{"id": wh.Id})
		r.Header.Set("X-Hub-Signature", "sha1=" + givenSignature)

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveValidSignature() should call next handler in chain")
			return
		}
	})
}

func Test_checkSHA1MAC(t *testing.T) {
	type args struct {
		message    []byte
		messageMAC []byte
		key        []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"valid hmac should pass",
			args{
				[]byte("Hello, World!"),
				func() []byte { res, _ := hex.DecodeString("816421f91f8bb65da114aef4616abf77052cccfe"); return res }(),
				[]byte("foobar"),
			},
			true,
		},
		{
			"invalid hmac should fail",
			args{
				[]byte("Hello, World!"),
				func() []byte { res, _ := hex.DecodeString("816421f91f8bb65da114aef4616abf77052cccfe"); return res }(),
				[]byte("foobar1234"),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkSHA1MAC(tt.args.message, tt.args.messageMAC, tt.args.key); got != tt.want {
				t.Errorf("checkSHA1MAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadRequestBodyHandler(t *testing.T) {
	t.Run("Request body should be put in context", func(t *testing.T) {
		body := "Hello, World!"

		nextHandlerCalled := false

		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true

			bodyFromContext := context.RequestBodyFromContext(r.Context())

			if body != string(bodyFromContext) {
				t.Errorf("ReadRequestBodyHandler() should set body in request context to <%v>, was <%v>", body, string(bodyFromContext))
				return
			}
		})

		handler := ReadRequestBodyHandler(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("ReadRequestBodyHandler() should call next handler in chain")
			return
		}
	})
}
