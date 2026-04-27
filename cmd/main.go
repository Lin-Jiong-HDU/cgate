package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lin-Jiong-HDU/go-project-template/bootstrap"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	app, err := bootstrap.Init()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := app.DB.Close(); closeErr != nil {
			slog.Error("close db", "error", closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.UC.Start(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", app.Server.Addr)
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-errCh:
		return err
	}

	app.UC.Stop()
	return app.Server.Shutdown(context.Background())
}
