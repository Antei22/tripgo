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
	"github.com/antei22/tripgo/internal/database"
	"github.com/antei22/tripgo/internal/httpserver"
	"github.com/antei22/tripgo/internal/trips"
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
	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: cfg.LogLevel,
			},
		),
	).With(
		"service",
		"trip-service",
	)

	slog.SetDefault(logger)

	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	pool, err := database.Open(appCtx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	// 5. Собираем зависимости приложения.
	repository := trips.NewRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	txManager := database.NewManager(
		pool,
		cfg.Database.QueryTimeout,
	)

	service := trips.NewService(
		repository,
		txManager,
	)

	handler := httpserver.NewHandler(
		service,
		pool,
		cfg.Database.QueryTimeout,
		logger,
	)

	router := httpserver.NewRouter(handler)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

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
