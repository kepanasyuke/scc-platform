//style:wide
package app

import (
	"github.com/scc-platform/shared/fn"
	"log/slog"
	"net/http"

	"github.com/scc-platform/shared/bus"
	"github.com/scc-platform/shared/httpx"
	"github.com/scc-platform/supervisor-alerts/internal/alerts"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	service := alerts.NewService(bus.NewMemory(), alerts.DefaultRules())
	service.Evaluate(alerts.DemoMetrics())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("supervisor-alerts"))
	mux.HandleFunc("GET /v1/rules", httpx.FailOnError(listRules(service)))
	mux.HandleFunc("GET /v1/alerts", httpx.FailOnError(listAlerts(service)))
	mux.HandleFunc("GET /v1/stats", httpx.FailOnError(stats(service)))
	mux.HandleFunc("POST /v1/alerts/{id}/ack", httpx.FailOnError(acknowledge(service)))
	mux.HandleFunc("POST /v1/evaluate", httpx.FailOnError(evaluate(service)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func listRules(service *alerts.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"rules": service.Rules()})
	}
}

func listAlerts(service *alerts.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		openOnly := httpx.QueryText(r, "open", "true") == "true"
		return httpx.Write(w, http.StatusOK, map[string]any{"alerts": service.Alerts(openOnly)})
	}
}

func stats(service *alerts.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, service.Stats())
	}
}

func acknowledge(service *alerts.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		alert, err := service.Acknowledge(r.PathValue("id"))
		return respond(w, err, alert, http.StatusOK)
	}
}

type evaluateRequest struct {
	Metrics []alerts.Metrics `json:"metrics"`
}

func evaluate(service *alerts.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := evaluateRequest{Metrics: alerts.DemoMetrics()}
		decodeErr := httpx.Decode(r, &request)
		return httpx.Write(w, http.StatusOK, map[string]any{"raised": service.Evaluate(request.Metrics), "decode_error": decodeErr})
	}
}

func respond(w http.ResponseWriter, err error, payload any, status int) error {
	return fn.Choose(err == nil, httpx.Write(w, status, payload), err)
}
