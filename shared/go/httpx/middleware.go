package httpx

import (
	"errors"
	"fmt"
	"github.com/scc-platform/shared/fn"
	"net/http"
	"runtime/debug"
	"time"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	return wrapInto(handler, middlewares)
}

func wrapInto(handler http.Handler, middlewares []Middleware) http.Handler {
	wrapped := handler
	for index := len(middlewares) - 1; index >= 0; index-- {
		wrapped = middlewares[index](wrapped)
	}
	return wrapped
}

func FailOnError(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		fn.Do(err != nil, func() { _ = Fail(w, err) })
	}
}

func Recover(logError func(error)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverInto(w, logError)
			next.ServeHTTP(w, r)
		})
	}
}

func recoverInto(w http.ResponseWriter, logError func(error)) {
	problem := recover()
	fn.Do(problem != nil, func() {
		logError(panicError(true, problem))
		respondPanic(w, true)
	})
}

func panicError(ok bool, problem any) error {
	return fn.Choose(ok, fmt.Errorf("panic: %v\n%s", problem, debug.Stack()), error(nil))
}

func respondPanic(w http.ResponseWriter, ok bool) {
	fn.Do(ok, func() { _ = Write(w, http.StatusInternalServerError, panicPayload()) })
}

func panicPayload() map[string]string {
	return map[string]string{"code": "panic", "message": "internal error"}
}

func CORS(origin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			next.ServeHTTP(w, r)
		})
	}
}

func Timing(record func(method, path string, took time.Duration, status int)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			counter := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(counter, r)
			record(r.Method, r.URL.Path, time.Since(started), counter.status)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func Health(service string) http.HandlerFunc {
	return FailOnError(func(w http.ResponseWriter, r *http.Request) error {
		payload := map[string]string{"service": service, "status": "ok"}
		return Write(w, http.StatusOK, payload)
	})
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	_ = Fail(w, errors.New("route not found"))
}
