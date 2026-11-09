package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/antei22/tripgo/internal/database"
	domain "github.com/antei22/tripgo/internal/domain/trip"
)

var sqlBuilder = squirrel.StatementBuilder.
	PlaceholderFormat(squirrel.Dollar)

var tripColumns = []string{
	"id",
	"user_id",
	"driver_id",
	"start_latitude",
	"start_longitude",
	"end_latitude",
	"end_longitude",
	"price",
	"status",
	"started_at",
	"finished_at",
}

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *Repository {
	return &Repository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (repo *Repository) Create(
	ctx context.Context,
	trip domain.Trip,
) error {
	query, args, err := sqlBuilder.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			string(trip.Status),
			trip.StartedAt,
			trip.FinishedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip query: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(
		ctx,
		repo.queryTimeout,
	)
	defer cancel()

	executor := database.Executor(queryCtx, repo.pool)

	_, err = executor.Exec(queryCtx, query, args...)
	if err == nil {
		return nil
	}

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) &&
		postgresError.Code == "23505" &&
		postgresError.ConstraintName ==
			"trips_one_active_per_driver_idx" {
		return domain.ErrDriverBusy
	}

	return fmt.Errorf("insert trip: %w", err)
}

func (repo *Repository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	fromStatus *domain.Status,
	toStatus domain.Status,
	reason *string,
) error {
	var fromStatusValue any
	if fromStatus != nil {
		fromStatusValue = string(*fromStatus)
	}

	query, args, err := sqlBuilder.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
		).
		Values(
			tripID,
			fromStatusValue,
			string(toStatus),
			reason,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf(
			"build insert status history query: %w",
			err,
		)
	}

	queryCtx, cancel := context.WithTimeout(
		ctx,
		repo.queryTimeout,
	)
	defer cancel()

	executor := database.Executor(queryCtx, repo.pool)

	if _, err := executor.Exec(queryCtx, query, args...); err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}

	return nil
}

func (repo *Repository) Get(
	ctx context.Context,
	tripID uuid.UUID,
) (domain.Trip, error) {
	query, args, err := sqlBuilder.
		Select(tripColumns...).
		From("trips").
		Where(squirrel.Eq{
			"id": tripID,
		}).
		ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf(
			"build select trip query: %w",
			err,
		)
	}

	queryCtx, cancel := context.WithTimeout(
		ctx,
		repo.queryTimeout,
	)
	defer cancel()

	executor := database.Executor(queryCtx, repo.pool)

	trip, err := scanTrip(
		executor.QueryRow(queryCtx, query, args...),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		return domain.Trip{}, fmt.Errorf("scan trip: %w", err)
	}

	return trip, nil
}

func (repo *Repository) GetForUpdate(ctx context.Context, tripID uuid.UUID) (domain.Trip, error) {
	query, args, err := sqlBuilder.Select(tripColumns...).From("trips").Where(squirrel.Eq{"id": tripID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build locked trip query: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, repo.queryTimeout)
	defer cancel()
	trip, err := scanTrip(database.Executor(queryCtx, repo.pool).QueryRow(queryCtx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Trip{}, domain.ErrTripNotFound
	}
	if err != nil {
		return domain.Trip{}, fmt.Errorf("fetch locked trip: %w", err)
	}
	return trip, nil
}

func (repo *Repository) SaveFinished(ctx context.Context, trip domain.Trip) error {
	query, args, err := sqlBuilder.Update("trips").Set("status", string(trip.Status)).Set("finished_at", trip.FinishedAt).Set("updated_at", trip.FinishedAt).Where(squirrel.Eq{"id": trip.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, repo.queryTimeout)
	defer cancel()
	_, err = database.Executor(queryCtx, repo.pool).Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(row rowScanner) (domain.Trip, error) {
	var trip domain.Trip
	var status string
	var finishedAt pgtype.Timestamptz

	err := row.Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&status,
		&trip.StartedAt,
		&finishedAt,
	)
	if err != nil {
		return domain.Trip{}, err
	}

	trip.Status = domain.Status(status)

	if finishedAt.Valid {
		value := finishedAt.Time
		trip.FinishedAt = &value
	}

	return trip, nil
}
