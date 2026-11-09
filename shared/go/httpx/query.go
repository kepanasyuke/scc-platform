package httpx

import (
	"net/http"
	"strconv"

	"github.com/scc-platform/shared/fn"
)

func QueryInt(r *http.Request, key string, fallback int) (int, error) {
	return parseInt(r.URL.Query().Get(key), fallback)
}

func parseInt(raw string, fallback int) (int, error) {
	parsed, err := strconv.Atoi(raw)
	return fn.Choose(raw == "", fallback, parsed), parseErr(raw == "", err)
}

func parseErr(absent bool, err error) error {
	return fn.Choose(absent, nil, err)
}

func QueryText(r *http.Request, key, fallback string) string {
	return fn.Text(r.URL.Query().Get(key) != "", r.URL.Query().Get(key), fallback)
}
