//style:wide
package app

import (
	"log/slog"
	"net/http"

	"github.com/scc-platform/pg-doctor/internal/pgdoctor"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("pg-doctor"))
	mux.HandleFunc("GET /v1/report", httpx.FailOnError(report()))
	mux.HandleFunc("GET /v1/samples", httpx.FailOnError(samples()))
	mux.HandleFunc("POST /v1/analyze", httpx.FailOnError(analyze()))
	mux.HandleFunc("POST /v1/deadlocks/parse", httpx.FailOnError(parseDeadlock()))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func report() httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		analysis := pgdoctor.Analyze(pgdoctor.DemoSamples(), pgdoctor.DemoDeadlocks())
		return httpx.Write(w, http.StatusOK, analysis)
	}
}

func samples() httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"samples": pgdoctor.DemoSamples()})
	}
}

type analyzeRequest struct {
	Samples []pgdoctor.Sample `json:"samples"`
}

func analyze() httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := analyzeRequest{Samples: pgdoctor.DemoSamples()}
		decodeErr := httpx.Decode(r, &request)
		analysis := pgdoctor.Analyze(request.Samples, pgdoctor.DemoDeadlocks())
		return analyzeDone(w, decodeErr, analysis)
	}
}

func analyzeDone(w http.ResponseWriter, decodeErr error, analysis pgdoctor.Analysis) error {
	return fn.Choose(decodeErr == nil, httpx.Write(w, http.StatusOK, analysis), decodeErr)
}

type logRequest struct {
	Entry string `json:"entry"`
}

func parseDeadlock() httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := logRequest{Entry: pgdoctor.DemoLog()}
		decodeErr := httpx.Decode(r, &request)
		return parsedDeadlock(w, decodeErr, pgdoctor.ParseDeadlockLog(request.Entry))
	}
}

func parsedDeadlock(w http.ResponseWriter, err error, deadlock pgdoctor.Deadlock) error {
	return fn.Choose(err == nil, httpx.Write(w, http.StatusOK, deadlock), err)
}
