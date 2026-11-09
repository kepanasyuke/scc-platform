//style:wide
package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
	"github.com/scc-platform/sip-probe/internal/sip"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	service := sip.NewService(sip.UDPProber{Timeout: 3 * time.Second}, sip.DefaultTrunks())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("sip-probe"))
	mux.HandleFunc("GET /v1/trunks", httpx.FailOnError(listTrunks(service)))
	mux.HandleFunc("GET /v1/trunks/last", httpx.FailOnError(lastProbes(service)))
	mux.HandleFunc("GET /v1/trunks/summary", httpx.FailOnError(probeSummary(service)))
	mux.HandleFunc("POST /v1/trunks/probe", httpx.FailOnError(probeAll(service)))
	mux.HandleFunc("POST /v1/trunks/{name}/probe", httpx.FailOnError(probeOne(service)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func listTrunks(service *sip.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"trunks": service.Trunks()})
	}
}

func lastProbes(service *sip.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"probes": service.Last()})
	}
}

func probeSummary(service *sip.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, service.Summary())
	}
}

func probeAll(service *sip.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"probes": service.ProbeAll(r.Context())})
	}
}

func probeOne(service *sip.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		result, err := service.Probe(r.Context(), r.PathValue("name"))
		return writeProbe(w, err, result)
	}
}

func writeProbe(w http.ResponseWriter, err error, result sip.ProbeResult) error {
	return httpx.Write(w, statusOf(err), result)
}

func statusOf(err error) int {
	return fn.Choose(err == nil, http.StatusOK, httpx.StatusOf(err))
}
