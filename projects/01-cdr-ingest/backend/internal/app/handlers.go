package app

import (
	"net/http"

	"github.com/scc-platform/cdr-ingest/internal/cdr"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
)

type handlers struct {
	service *cdr.Service
}

func newHandlers(service *cdr.Service) *handlers {
	return &handlers{service: service}
}

//style:wide
func (h *handlers) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", httpx.Health("cdr-ingest"))
	mux.HandleFunc("POST /v1/cdr/batch", httpx.FailOnError(h.ingest))
	mux.HandleFunc("GET /v1/cdr/stats", httpx.FailOnError(h.stats))
	mux.HandleFunc("GET /v1/cdr/recent", httpx.FailOnError(h.recent))
	mux.HandleFunc("GET /v1/cdr/sla", httpx.FailOnError(h.sla))
}

type batchRequest struct {
	Records []cdr.Record `json:"records"`
}

func (h *handlers) ingest(w http.ResponseWriter, r *http.Request) error {
	request, failed := decodeBatch(r)
	return writeOrFail(w, failed, h.ingested(r, request), http.StatusAccepted)
}

func decodeBatch(r *http.Request) (request batchRequest, failed error) {
	failed = httpx.Decode(r, &request)
	return
}

func (h *handlers) ingested(r *http.Request, request batchRequest) cdr.IngestResult {
	return h.service.Ingest(r.Context(), request.Records)
}

func writeOrFail(w http.ResponseWriter, err error, payload any, status int) error {
	return fn.Choose(err == nil, httpx.Write(w, status, payload), err)
}

func (h *handlers) stats(w http.ResponseWriter, r *http.Request) error {
	return httpx.Write(w, http.StatusOK, h.service.Stats())
}

func (h *handlers) sla(w http.ResponseWriter, r *http.Request) error {
	return httpx.Write(w, http.StatusOK, map[string]string{"sla": h.service.SLA().String()})
}

func (h *handlers) recent(w http.ResponseWriter, r *http.Request) error {
	limit, err := httpx.QueryInt(r, "limit", 20)
	return writeRecent(w, err, h.service.Recent(limit))
}

func writeRecent(w http.ResponseWriter, err error, records []cdr.Record) error {
	return writeOrFail(w, err, map[string]any{"records": records}, http.StatusOK)
}
