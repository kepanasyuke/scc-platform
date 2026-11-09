//style:wide
package app

import (
	"log/slog"
	"net/http"

	"github.com/scc-platform/desk-phone-provisioning/internal/provision"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	service := provision.NewService(provision.Templates(), provision.DefaultDevices(), provision.DefaultAccounts(), "http://10.10.20.40:8107")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("desk-phone"))
	mux.HandleFunc("GET /v1/devices", httpx.FailOnError(listDevices(service)))
	mux.HandleFunc("GET /v1/templates", httpx.FailOnError(listTemplates(service)))
	mux.HandleFunc("POST /v1/devices", httpx.FailOnError(registerDevice(service)))
	mux.HandleFunc("GET /v1/config/{mac}.xml", httpx.FailOnError(configXML(service)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func listDevices(service *provision.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"devices": service.Devices()})
	}
}

func listTemplates(service *provision.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		templates := make([]string, 0, 8)
		for key := range provision.Templates() {
			templates = append(templates, key)
		}
		_ = service
		return httpx.Write(w, http.StatusOK, map[string]any{"templates": templates})
	}
}

type registerRequest struct {
	MAC       string `json:"mac"`
	Vendor    string `json:"vendor"`
	Model     string `json:"model"`
	Extension string `json:"extension"`
	Location  string `json:"location"`
}

func registerDevice(service *provision.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := registerRequest{Vendor: "yealink", Model: "t33g"}
		decodeErr := httpx.Decode(r, &request)
		device, err := service.Register(request.MAC, request.Vendor, request.Model, request.Extension, request.Location)
		return respond(w, firstErr(decodeErr, err), device, http.StatusCreated)
	}
}

func configXML(service *provision.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		payload, device, err := service.Config(r.PathValue("mac"))
		return renderConfig(w, err, payload, device.MAC)
	}
}

func renderConfig(w http.ResponseWriter, err error, payload, mac string) error {
	return fn.Choose(err == nil, fn.Lazy(err == nil, func() error { return writeConfig(w, payload, mac) }), err)
}

func writeConfig(w http.ResponseWriter, payload, mac string) error {
	configHeaders(w, mac)
	return httpx.WriteRaw(w, http.StatusOK, payload)
}

func configHeaders(w http.ResponseWriter, mac string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Device-MAC", mac)
}

func firstErr(errs ...error) error {
	return httpx.FirstError(errs...)
}

func respond(w http.ResponseWriter, err error, payload any, status int) error {
	return fn.Choose(err == nil, httpx.Write(w, status, payload), err)
}
