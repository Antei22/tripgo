package trips

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/antei22/tripgo/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	trip Trip,
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
		return ErrDriverBusy
	}

	return fmt.Errorf("insert trip: %w", err)
}

func (repo *Repository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	fromStatus *Status,
	toStatus Status,
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
) (Trip, error) {
	query, args, err := sqlBuilder.
		Select(tripColumns...).
		From("trips").
		Where(squirrel.Eq{
			"id": tripID,
		}).
		ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf(
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
		return Trip{}, ErrTripNotFound
	}
	if err != nil {
		return Trip{}, fmt.Errorf("scan trip: %w", err)
	}

	return trip, nil
}

func (repo *Repository) FinishActive(
	ctx context.Context,
	tripID uuid.UUID,
	finishedAt time.Time,
) (Trip, error) {
	query, args, err := sqlBuilder.
		Update("trips").
		Set("status", string(StatusCompleted)).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{
			"id":     tripID,
			"status": string(StatusActive),
		}).
		Suffix(
			"RETURNING " + strings.Join(tripColumns, ", "),
		).
		ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf(
			"build finish trip query: %w",
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
	if err == nil {
		return trip, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, fmt.Errorf("update trip status: %w", err)
	}

	existingTrip, getErr := repo.Get(ctx, tripID)
	if getErr != nil {
		return Trip{}, getErr
	}

	if existingTrip.Status == StatusCompleted {
		return Trip{}, ErrTripCompleted
	}

	return Trip{}, fmt.Errorf(
		"trip %s was not updated from status %q",
		tripID,
		existingTrip.Status,
	)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(row rowScanner) (Trip, error) {
	var trip Trip
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
		return Trip{}, err
	}

	trip.Status = Status(status)

	if finishedAt.Valid {
		value := finishedAt.Time
		trip.FinishedAt = &value
	}

	return trip, nil
}
