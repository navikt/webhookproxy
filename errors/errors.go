package errors

import (
	"net/http"
	"encoding/json"
)

type appError struct {
	status  int
	Message string `json:"message"`
}

func (a appError) Status() int {
	return a.status
}

func (a appError) Error() string {
	b, _ := json.Marshal(a)
	return string(b)
}

func NewAppError(status int, msg string) appError {
	return appError{status, msg}
}

func RespondWithError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch err := err.(type) {
	case appError:
		status = err.Status()
	}
	http.Error(w, err.Error(), status)
}
