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
	"github.com/gorilla/mux"
	"github.com/navikt/webhookproxy/errors"
)

type Middleware func(http.Handler) http.Handler

func MustHaveWebhook(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		webhookId := vars["id"]

		wh := webhook.Get(webhookId)

		if wh == nil {
			fmt.Fprintf(os.Stderr, "webhook does not exist: %v\n", webhookId)
			errors.RespondWithError(w, errors.NewAppError(http.StatusNotFound, "webhook does not exist"))
			return
		}

		ctx := context.NewContextWithWebhook(r.Context(), wh)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MustHaveValidSignature(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signatureHeader := r.Header.Get("X-Hub-Signature")

		signatureInfo := strings.Split(signatureHeader, "=")
		if len(signatureInfo) != 2 {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v\n", signatureInfo)
			errors.RespondWithError(w, errors.NewAppError(http.StatusBadRequest, "malformed signature header"))
			return
		}

		if signatureInfo[0] != "sha1" {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v: unknown algo: %v\n", signatureHeader, signatureInfo[0])
			errors.RespondWithError(w, errors.NewAppError(http.StatusBadRequest, "malformed signature header, unknown algo"))
			return
		}

		signature, err := hex.DecodeString(signatureInfo[1])

		if err != nil {
			fmt.Fprintf(os.Stderr,"invalid signature header: %v\n", err)
			errors.RespondWithError(w, errors.NewAppError(http.StatusBadRequest, "malformed signature header, unknown contents"))
			return
		}

		if !checkSHA1MAC(context.RequestBodyFromContext(r.Context()), signature, context.WebhookFromContext(r.Context()).Secret) {
			fmt.Fprintf(os.Stderr,"invalid signature: %x\n", signature)
			errors.RespondWithError(w, errors.NewAppError(http.StatusForbidden, "invalid signature"))
			return
		}

		h.ServeHTTP(w, r)
	})
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
			errors.RespondWithError(w, errors.NewAppError(http.StatusInternalServerError, "failed to read request body"))
			return
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
