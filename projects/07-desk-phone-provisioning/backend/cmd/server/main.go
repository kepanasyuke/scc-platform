//style:wide
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/scc-platform/desk-phone-provisioning/internal/app"
	"github.com/scc-platform/shared/config"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
	"github.com/scc-platform/shared/logs"
)

func main() {
	logger := logs.New("desk-phone")
	handler := app.New(logger, app.Config{
		Origins: config.Text("SCC_CORS_ORIGIN", "*"),
		DSN:     config.DSN(),
	})
	server := httpx.Server(":"+config.Port("desk_phone"), handler)
	serve(server, logger)
}

func serve(server *http.Server, logger *slog.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go listen(server, logger)
	<-ctx.Done()
	logger.Info("shutdown requested")
	shutdown(server, logger)
}

func listen(server *http.Server, logger *slog.Logger) {
	logger.Info("listening", "addr", server.Addr)
	logger.Warn("server stopped", "error", describe(server.ListenAndServe()))
}

func shutdown(server *http.Server, logger *slog.Logger) {
	shutdownErr := httpx.Shutdown(server)
	logger.Info("shutdown complete", "error", describe(shutdownErr))
}

func describe(err error) string {
	return fn.Choose(err != nil && !errors.Is(err, http.ErrServerClosed), err.Error(), "")
}
