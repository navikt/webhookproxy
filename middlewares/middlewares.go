package middlewares

import (
	"strings"
	"github.com/navikt/webhookproxy/webhook"
	"fmt"
	"os"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"time"
	"encoding/hex"
	"github.com/navikt/webhookproxy/context"
)

type Middleware func(http.Handler) http.Handler

func MustHaveHeader(headers ...string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range headers {
				if r.Header.Get(header) == "" {
					http.Error(w, "missing required header "+header, http.StatusBadRequest)
					return
				}
			}

			h.ServeHTTP(w, r)
		})
	}
}

func MustHaveMethod(method string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				http.Error(w, "expected " + method, http.StatusMethodNotAllowed)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

func MustHaveSignature(h http.Handler) http.Handler {
	return MustHaveHeader("X-Hub-Signature")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signatureHeader := r.Header.Get("X-Hub-Signature")

		signatureInfo := strings.Split(signatureHeader, "=")
		if len(signatureInfo) != 2 {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v\n", signatureInfo)
			http.Error(w, "malformed signature header", http.StatusBadRequest)
			return
		}

		if signatureInfo[0] != "sha1" {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v: unknown algo: %v\n", signatureHeader, signatureInfo[0])
			http.Error(w, "malformed signature header, unknown algo", http.StatusBadRequest)
			return
		}

		signature, err := hex.DecodeString(signatureInfo[1])

		if err != nil {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v\n", err)
			http.Error(w, "malformed signature header, unknown contents", http.StatusBadRequest)
			return
		}

		ctx := context.NewContextWithSignature(r.Context(), signature)
		h.ServeHTTP(w,  r.WithContext(ctx))
	}))
}

func MustHaveWebhook(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookId := strings.Replace(r.URL.Path, "/hook/", "", 1)

		wh := webhook.Get(webhookId)

		if wh == nil {
			fmt.Fprintf(os.Stderr, "webhook does not exist: %v\n", webhookId)
			http.Error(w, "webhook does not exist", http.StatusNotFound)
			return
		}

		ctx := context.NewContextWithWebhook(r.Context(), wh)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MustHaveValidSignature(h http.Handler) http.Handler {
	return MustHaveSignature(MustHaveWebhook(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkSHA1MAC(context.RequestBodyFromContext(r.Context()), context.SignatureFromContext(r.Context()), context.WebhookFromContext(r.Context()).Secret) {
			fmt.Fprintf(os.Stderr,"invalid signature: %x\n", context.SignatureFromContext(r.Context()))
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})))
}

func checkSHA1MAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}

func LogHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%v][%v] -> %v %v %v\n", time.Now(), r.RemoteAddr, r.Proto, r.Method, r.URL)
		fmt.Printf("Headers:\n")
		for key, val := range r.Header {
			fmt.Printf("%v: %v\n", key, val)
		}

		h.ServeHTTP(w, r)
	})
}

func ReadRequestBodyHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := context.NewContextWithRequestBody(r.Context(), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read request body: %v\n", err)
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
