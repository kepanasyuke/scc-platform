package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/scc-platform/shared/fn"
)

const defaultPort = "8080"

func Text(key, fallback string) string {
	return pick(os.Getenv(key), fallback)
}

func pick(value, fallback string) string {
	clean := strings.TrimSpace(value)
	return fn.Text(clean != "", clean, fallback)
}

func Port(service string) string {
	return Text("SCC_"+strings.ToUpper(service)+"_PORT", defaultPort)
}

func Int(key string, fallback int) int {
	return parseInt(Text(key, ""), fallback)
}

func parseInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(raw)
	return fn.Number(err == nil, parsed, fallback)
}

func Duration(key string, fallback time.Duration) time.Duration {
	return parseDuration(Text(key, ""), fallback)
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(raw)
	return fn.Choose(err == nil, parsed, fallback)
}

func DSN() string {
	return Text("SCC_PG_DSN", "postgres://scc:scc@localhost:5432/scc?sslmode=disable")
}

func NATSURL() string {
	return Text("SCC_NATS_URL", "nats://localhost:4222")
}
