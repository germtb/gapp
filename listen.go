package gapp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ListenAndServe starts an HTTP server and blocks until a SIGINT or SIGTERM
// signal is received, at which point it initiates a graceful shutdown with a
// 30-second timeout. Returns http.ErrServerClosed on clean shutdown.
func ListenAndServe(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("Server starting", "addr", addr)
		errCh <- server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		// Server failed to start or crashed
		return err
	case sig := <-sigCh:
		slog.Info("Shutdown signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Info("Shutting down gracefully...")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed", "error", err)
		return err
	}

	slog.Info("Server stopped")
	return http.ErrServerClosed
}
