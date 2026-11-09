package container

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/antei22/tripgo/internal/config"
	"github.com/antei22/tripgo/internal/database"
	"github.com/antei22/tripgo/internal/httpserver"
	"github.com/antei22/tripgo/internal/infra/telemetry"
	repository "github.com/antei22/tripgo/internal/repository/trip"
	create "github.com/antei22/tripgo/internal/usecase/trip/create"
	finish "github.com/antei22/tripgo/internal/usecase/trip/finish"
	get "github.com/antei22/tripgo/internal/usecase/trip/get"
)

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*http.Server, *pgxpool.Pool, error) {
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	if err := telemetry.RegisterActiveTrips(pool, cfg.Database.QueryTimeout); err != nil {
		pool.Close()
		return nil, nil, err
	}
	repo := repository.NewRepository(pool, cfg.Database.QueryTimeout)
	tx := database.NewManager(pool, cfg.Database.QueryTimeout)
	creator := create.NewService(repo, tx)
	getter := get.NewService(repo)
	finisher := finish.NewService(repo, tx)
	handler := httpserver.NewHandler(creator, getter, finisher, pool, cfg.Database.QueryTimeout, logger)
	server := &http.Server{Addr: cfg.HTTP.Addr, Handler: httpserver.NewRouter(handler), ReadTimeout: cfg.HTTP.ReadTimeout, ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
	return server, pool, nil
}
