package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/scc-platform/shared/fn"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e Error) Error() string {
	return e.Code + ": " + e.Message
}

func BadRequest(message string) error {
	return Error{Code: "bad_request", Message: message, Status: http.StatusBadRequest}
}

func NotFound(what string) error {
	return Error{Code: "not_found", Message: what + " not found", Status: http.StatusNotFound}
}

func Unavailable(message string) error {
	return Error{Code: "unavailable", Message: message, Status: http.StatusServiceUnavailable}
}

func StatusOf(err error) int {
	var api Error
	return fn.Choose(errors.As(err, &api), api.Status, http.StatusInternalServerError)
}

func Write(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return encodeInto(w, status, payload)
}

func encodeInto(w http.ResponseWriter, status int, payload any) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func Fail(w http.ResponseWriter, err error) error {
	api := asError(err)
	return Write(w, api.Status, map[string]string{"code": api.Code, "message": api.Message})
}

func asError(err error) Error {
	var api Error
	return fn.Choose(errors.As(err, &api), api, internalError())
}

func internalError() Error {
	return Error{Code: "internal", Message: "internal error", Status: http.StatusInternalServerError}
}

func Decode(r *http.Request, target any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(target)
}
