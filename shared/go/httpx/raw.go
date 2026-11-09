package httpx

import (
	"net/http"
)

func WriteRaw(w http.ResponseWriter, status int, body string) error {
	return writeBody(w, status, body)
}

func writeBody(w http.ResponseWriter, status int, body string) error {
	w.WriteHeader(status)
	return written(w, body)
}

func written(w http.ResponseWriter, body string) error {
	_, err := w.Write([]byte(body))
	return err
}
