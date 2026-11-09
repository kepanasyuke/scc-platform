import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";

const root = process.cwd();

const services = [
  { dir: "01-cdr-ingest", module: "cdr-ingest", name: "cdr-ingest", port: "cdr_ingest" },
  { dir: "02-call-analytics", module: "call-analytics", name: "call-analytics", port: "call_analytics" },
  { dir: "03-sip-probe", module: "sip-probe", name: "sip-probe", port: "sip_probe" },
  { dir: "04-queue-acd", module: "queue-acd", name: "queue-acd", port: "queue_acd" },
  { dir: "05-supervisor-alerts", module: "supervisor-alerts", name: "supervisor-alerts", port: "supervisor_alerts" },
  { dir: "06-freeswitch-esl", module: "freeswitch-esl", name: "freeswitch-esl", port: "freeswitch_esl" },
  { dir: "07-desk-phone-provisioning", module: "desk-phone-provisioning", name: "desk-phone", port: "desk_phone" },
  { dir: "08-pg-doctor", module: "pg-doctor", name: "pg-doctor", port: "pg_doctor" }
];

const main = (service) => `//style:wide
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/scc-platform/${service.module}/internal/app"
	"github.com/scc-platform/shared/config"
	"github.com/scc-platform/shared/httpx"
	"github.com/scc-platform/shared/logs"
)

func main() {
	logger := logs.New("${service.name}")
	handler := app.New(logger, app.Config{
		Origins: config.Text("SCC_CORS_ORIGIN", "*"),
		DSN:     config.DSN(),
	})
	server := httpx.Server(":"+config.Port("${service.port}"), handler)
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
	text := ""
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		text = err.Error()
	}
	return text
}
`;

for (const service of services) {
  const target = join(root, "projects", service.dir, "backend", "cmd", "server");
  await mkdir(target, { recursive: true });
  await writeFile(join(target, "main.go"), main(service), "utf8");
  process.stdout.write(`${service.dir}/backend/cmd/server/main.go\n`);
}
