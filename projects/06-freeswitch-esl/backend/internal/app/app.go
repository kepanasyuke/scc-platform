//style:wide
package app

import (
	"log/slog"
	"net/http"

	"github.com/scc-platform/freeswitch-esl/internal/esl"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	sw := esl.NewMemorySwitch()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("freeswitch-esl"))
	mux.HandleFunc("GET /v1/esl/status", httpx.FailOnError(status(sw)))
	mux.HandleFunc("GET /v1/channels", httpx.FailOnError(channels(sw)))
	mux.HandleFunc("GET /v1/channels/summary", httpx.FailOnError(summary(sw)))
	mux.HandleFunc("POST /v1/channels/{uuid}/hangup", httpx.FailOnError(hangup(sw)))
	mux.HandleFunc("POST /v1/originate", httpx.FailOnError(originate(sw)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func status(sw esl.Switch) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, sw.Status())
	}
}

func channels(sw esl.Switch) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"channels": sw.Channels()})
	}
}

func summary(sw esl.Switch) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, esl.Summarize(sw.Channels()))
	}
}

func hangup(sw esl.Switch) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		channel, err := sw.Hangup(r.PathValue("uuid"))
		return respond(w, err, channel, http.StatusOK)
	}
}

type originateRequest struct {
	Destination string `json:"destination"`
	Extension   string `json:"extension"`
}

func originate(sw esl.Switch) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := originateRequest{Extension: "104"}
		decodeErr := httpx.Decode(r, &request)
		channel, err := sw.Originate(request.Destination, request.Extension)
		return respond(w, firstErr(decodeErr, err), channel, http.StatusCreated)
	}
}

func firstErr(errs ...error) error {
	return httpx.FirstError(errs...)
}

func respond(w http.ResponseWriter, err error, payload any, status int) error {
	return fn.Choose(err == nil, httpx.Write(w, status, payload), err)
}
