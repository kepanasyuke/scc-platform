package httpx

import (
	"context"
	"net/http"
	"time"
)

func Server(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func Shutdown(server *http.Server) error {
	ctx, cancel := timeout()
	return shutDown(server, ctx, cancel)
}

func timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func shutDown(server *http.Server, ctx context.Context, cancel context.CancelFunc) error {
	defer cancel()
	return server.Shutdown(ctx)
}
