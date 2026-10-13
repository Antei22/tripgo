package trips

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type transactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repository *Repository
	txManager  transactionManager
}

func NewService(
	repository *Repository,
	txManager transactionManager,
) *Service {
	return &Service{
		repository: repository,
		txManager:  txManager,
	}
}

func (service *Service) Create(
	ctx context.Context,
	input CreateInput,
) (Trip, error) {
	trip := Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     StatusActive,
		StartedAt:  time.Now().UTC(),
	}

	err := service.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := service.repository.Create(txCtx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := service.repository.AddStatusHistory(
			txCtx,
			trip.ID,
			nil,
			StatusActive,
			nil,
		); err != nil {
			return fmt.Errorf("add initial status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return Trip{}, err
	}

	return trip, nil
}

func (service *Service) Get(
	ctx context.Context,
	tripID uuid.UUID,
) (Trip, error) {
	return service.repository.Get(ctx, tripID)
}

func (service *Service) Finish(
	ctx context.Context,
	tripID uuid.UUID,
) (Trip, error) {
	var result Trip

	err := service.txManager.Do(ctx, func(txCtx context.Context) error {
		finishedAt := time.Now().UTC()

		trip, err := service.repository.FinishActive(
			txCtx,
			tripID,
			finishedAt,
		)
		if err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		fromStatus := StatusActive
		if err := service.repository.AddStatusHistory(
			txCtx,
			tripID,
			&fromStatus,
			StatusCompleted,
			nil,
		); err != nil {
			return fmt.Errorf("add completed status history: %w", err)
		}

		result = trip
		return nil
	})
	if err != nil {
		return Trip{}, err
	}

	return result, nil
}
