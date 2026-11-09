package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/antei22/tripgo/internal/config"
	"github.com/antei22/tripgo/internal/container"
	"github.com/antei22/tripgo/internal/infra/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error(
			"service stopped with error",
			"error",
			err,
		)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Настраиваем JSON-логи.
	logger, shutdownTelemetry, err := telemetry.Setup(context.Background(), cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("setup telemetry: %w", err)
	}
	var closePool func()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			logger.Error("telemetry shutdown failed", "error", err)
		}
		if closePool != nil {
			closePool()
		}
	}()

	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	server, pool, err := container.New(appCtx, cfg, logger)
	if err != nil {
		return fmt.Errorf("initialize container: %w", err)
	}
	closePool = pool.Close

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info(
			"HTTP server started",
			"address",
			cfg.HTTP.Addr,
		)

		err := server.ListenAndServe()

		if err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}

		serverErrors <- nil
	}()

	select {
	case serverErr := <-serverErrors:
		if serverErr != nil {
			return fmt.Errorf(
				"serve HTTP: %w",
				serverErr,
			)
		}

		return nil

	case <-appCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf(
			"shutdown HTTP server: %w",
			err,
		)
	}

	if serverErr := <-serverErrors; serverErr != nil {
		return fmt.Errorf(
			"HTTP server stopped: %w",
			serverErr,
		)
	}

	logger.Info("service stopped gracefully")

	return nil
}
