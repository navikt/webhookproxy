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
)

func TestMustHaveHeader(t *testing.T) {
	t.Run("Empty headers should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveHeader() should not call next handler in chain")
		})

		middleware := MustHaveHeader("X-Header")
		handler := middleware(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("MustHaveHeader() should set http code to %v", http.StatusBadRequest)
			return
		}

		if w.Body.String() != "missing required header X-Header\n" {
			t.Errorf("MustHaveHeader() should set body to <%v>, was <%v>", "missing required header X-Header", w.Body.String())
			return
		}
	})
	t.Run("Non-empty headers should pass", func(t *testing.T) {
		nextHandlerCalled := false
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
		})

		middleware := MustHaveHeader("X-Header")
		handler := middleware(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("X-Header", "foo")

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveHeader() should call next handler in chain")
			return
		}

		if w.Code != http.StatusOK {
			t.Errorf("MustHaveHeader() should not set http code to %v", w.Code)
			return
		}
	})
}

func TestMustHaveMethod(t *testing.T) {
	for _, method := range []string{"GET", "OPTIONS", "HEAD"} {
		t.Run("Method "+method+" is not POST and should fail", func(t *testing.T) {
			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("MustHaveHeader() should not call next handler in chain")
			})

			middleware := MustHaveMethod("POST")
			handler := middleware(dummyHandler)

			w := httptest.NewRecorder()

			r := httptest.NewRequest(method, "/", nil)

			handler.ServeHTTP(w, r)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("MustHaveMethod() should set http code to %v", http.StatusMethodNotAllowed)
				return
			}

			if w.Body.String() != "expected POST\n" {
				t.Errorf("MustHaveMethod() should set body to <%v>, was <%v>", "expected POST", w.Body.String())
				return
			}
		})
	}

	t.Run("Method POST should pass", func(t *testing.T) {
		nextHandlerCalled := false
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
		})

		middleware := MustHaveMethod("POST")
		handler := middleware(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveMethod() should call next handler in chain")
			return
		}

		if w.Code != http.StatusOK {
			t.Errorf("MustHaveMethod() should not set http code to %v", w.Code)
			return
		}
	})
}

func TestMustHaveSignature(t *testing.T) {
	t.Run("Empty headers should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("MustHaveSignature() should set http code to %v", http.StatusBadRequest)
			return
		}

		if w.Body.String() != "missing required header X-Hub-Signature\n" {
			t.Errorf("MustHaveSignature() should set body to <%v>, was <%v>", "missing required header X-Hub-Signature", w.Body.String())
			return
		}
	})
	t.Run("Invalid header should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("X-Hub-Signature", "foo")

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("MustHaveSignature() should set http code to %v", http.StatusBadRequest)
			return
		}

		if w.Body.String() != "malformed signature header\n" {
			t.Errorf("MustHaveSignature() should set body to <%v>, was <%v>", "malformed signature header", w.Body.String())
			return
		}
	})
	t.Run("Invalid algo should fail", func(t *testing.T) {
		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			t.Errorf("MustHaveSignature() should not call next handler in chain")
		})

		handler := MustHaveSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("X-Hub-Signature", "sha2=aebfc")

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("MustHaveSignature() should set http code to %v", http.StatusBadRequest)
			return
		}

		if w.Body.String() != "malformed signature header, unknown algo\n" {
			t.Errorf("MustHaveSignature() should set body to <%v>, was <%v>", "malformed signature header, unknown algo", w.Body.String())
			return
		}
	})
	t.Run("Signature should be put in context", func(t *testing.T) {
		givenSignature := "413a5a5e9bf8651218acaffb55a66b06f9b6eb50"
		nextHandlerCalled := false

		dummyHandler := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true

			signature := context.SignatureFromContext(r.Context())

			if hex.EncodeToString(signature) != givenSignature {
				t.Errorf("MustHaveSignature() should set signature in request context to <%v>, was <%v>", givenSignature, hex.EncodeToString(signature))
				return
			}
		})

		handler := MustHaveSignature(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)

		r.Header.Set("X-Hub-Signature", "sha1=" + givenSignature)

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveSignature() should call next handler in chain")
			return
		}
	})
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

		if w.Code != http.StatusNotFound {
			t.Errorf("MustHaveWebhook() should set http code to %v", http.StatusNotFound)
			return
		}

		if w.Body.String() != "webhook does not exist\n" {
			t.Errorf("MustHaveWebhook() should set body to <%v>, was <%v>", "webhook does not exist", w.Body.String())
			return
		}
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

		handler := MustHaveWebhook(dummyHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, nil)

		handler.ServeHTTP(w, r)

		if !nextHandlerCalled {
			t.Errorf("MustHaveWebhook() should call next handler in chain")
			return
		}
	})
}

func TestMustHaveValidSignature(t *testing.T) {
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

		handler := ReadRequestBodyHandler(MustHaveValidSignature(dummyHandler))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, strings.NewReader("Hello, World! This message has been altered."))

		r.Header.Set("X-Hub-Signature", "sha1=" + givenSignature)

		handler.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("MustHaveValidSignature() should set http code to %v", http.StatusForbidden)
			return
		}

		if w.Body.String() != "invalid signature\n" {
			t.Errorf("MustHaveValidSignature() should set body to <%v>, was <%v>", "invalid signature", w.Body.String())
			return
		}
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

		handler := ReadRequestBodyHandler(MustHaveValidSignature(dummyHandler))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/hook/" + wh.Id, strings.NewReader("Hello, World!"))

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
