//style:wide
package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/scc-platform/queue-acd/internal/acd"
	"github.com/scc-platform/shared/bus"
	"github.com/scc-platform/shared/config"
	"github.com/scc-platform/shared/fn"
	"github.com/scc-platform/shared/httpx"
)

type Config struct {
	Origins string
	DSN     string
}

//style:wide
func New(logger *slog.Logger, cfg Config) http.Handler {
	hub := bus.NewMemory()
	service := acd.NewService(hub, config.Duration("SCC_ACD_SLA", 20*time.Second), acd.DefaultAgents(), acd.DefaultQueues())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpx.Health("queue-acd"))
	mux.HandleFunc("GET /v1/queues", httpx.FailOnError(allQueues(service)))
	mux.HandleFunc("GET /v1/queues/{name}", httpx.FailOnError(oneQueue(service)))
	mux.HandleFunc("POST /v1/queues/{name}/enqueue", httpx.FailOnError(enqueue(service)))
	mux.HandleFunc("POST /v1/queues/{name}/dispatch", httpx.FailOnError(dispatch(service)))
	mux.HandleFunc("GET /v1/agents", httpx.FailOnError(listAgents(service)))
	mux.HandleFunc("POST /v1/agents/{ext}/state", httpx.FailOnError(changeAgentState(service)))
	recoverer := httpx.Recover(func(err error) { logger.Error("panic recovered", "error", err) })
	return httpx.Chain(mux, recoverer, httpx.CORS(cfg.Origins))
}

func allQueues(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		names := service.Queues()
		snapshots := make([]acd.Snapshot, 0, len(names))
		for _, name := range names {
			snapshots = append(snapshots, service.Snapshot(name))
		}
		return httpx.Write(w, http.StatusOK, map[string]any{"queues": snapshots})
	}
}

func oneQueue(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, service.Snapshot(r.PathValue("name")))
	}
}

type enqueueRequest struct {
	Caller   string `json:"caller"`
	Skill    string `json:"skill"`
	Priority int    `json:"priority"`
}

func enqueue(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := enqueueRequest{Skill: "general"}
		decodeErr := httpx.Decode(r, &request)
		return enqueued(w, r, service, request, decodeErr)
	}
}

func enqueued(w http.ResponseWriter, r *http.Request, service *acd.Service, request enqueueRequest, decodeErr error) error {
	ticket, err := service.Enqueue(r.Context(), r.PathValue("name"), request.Caller, request.Skill, request.Priority)
	return respond(w, firstErr(decodeErr, err), map[string]any{"ticket": ticket}, http.StatusCreated)
}

func dispatch(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		assignment, err := service.Dispatch(r.Context(), r.PathValue("name"))
		return respond(w, err, assignment, http.StatusOK)
	}
}

func listAgents(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return httpx.Write(w, http.StatusOK, map[string]any{"agents": service.Agents()})
	}
}

type stateRequest struct {
	State string `json:"state"`
}

func changeAgentState(service *acd.Service) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		request := stateRequest{State: "ready"}
		decodeErr := httpx.Decode(r, &request)
		return stateChanged(w, service, r.PathValue("ext"), request, decodeErr)
	}
}

func stateChanged(w http.ResponseWriter, service *acd.Service, ext string, request stateRequest, decodeErr error) error {
	agent, err := service.SetAgentState(ext, request.State)
	return respond(w, firstErr(decodeErr, err), agent, http.StatusOK)
}

func firstErr(errs ...error) error {
	return httpx.FirstError(errs...)
}

func respond(w http.ResponseWriter, err error, payload any, status int) error {
	return fn.Choose(err == nil, httpx.Write(w, status, payload), err)
}
