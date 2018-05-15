package main

import (
	"net/http"
	"fmt"
	"os"
	"io/ioutil"
	"io"
)

type server struct {}

type delegatedWriter struct {
	writers []io.Writer
}

func (d delegatedWriter) Write(p []byte) (n int, err error) {
	for _, writer := range d.writers {
		n, err := writer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (s *server) handler(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(200)
	w.Header().Set("content-type", "text/plain")

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return err
	}

	dw := delegatedWriter{[]io.Writer{os.Stdout, w}}

	fmt.Fprintf(dw, "%v %v %v\n", r.Proto, r.Method, r.URL)
	fmt.Fprintf(dw, "Remote addr: %v\n", r.RemoteAddr)
	fmt.Fprintln(dw, "Headers:")
	for key, val := range r.Header {
		fmt.Fprintf(dw, "%v: %v\n", key, val)
	}
	fmt.Fprintln(dw, "\nBody:")
	dw.Write(body)

	return nil
}

type httpErrorHandlerWrapper func(w http.ResponseWriter, r *http.Request) error

func (fn httpErrorHandlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	autoIndex := &server{}
	http.Handle("/", httpErrorHandlerWrapper(autoIndex.handler))
	panic(http.ListenAndServe(listenAddr, nil))
}
