//style:wide
package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/scc-platform/cdr-ingest/internal/cdr"
	"github.com/scc-platform/shared/config"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	service := cdr.NewService(config.Duration("SCC_CDR_SLA", 20*time.Second))
	mux := http.NewServeMux()
	newHandlers(service).register(mux)
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}
