//style:wide
package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/scc-platform/call-analytics/internal/analytics"
	"github.com/scc-platform/shared/config"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	service := analytics.NewService(analytics.SeedCalls(config.Int("SCC_ANALYTICS_SAMPLE", 1600), seedDay()), 20*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("call-analytics"))
	mux.HandleFunc("GET /v1/analytics/hourly", httpx.FailOnError(hourly(service)))
	mux.HandleFunc("GET /v1/analytics/queues", httpx.FailOnError(queues(service)))
	mux.HandleFunc("GET /v1/analytics/summary", httpx.FailOnError(summary(service)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func seedDay() time.Time {
	return time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
}

func hourly(service *analytics.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"buckets": service.Hourly()})
	}
}

func queues(service *analytics.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"queues": service.Queues()})
	}
}

func summary(service *analytics.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, service.Summary())
	}
}
