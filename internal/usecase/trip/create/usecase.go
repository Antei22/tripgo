package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/antei22/tripgo/internal/domain/trip"
)

type Service struct {
	repository tripRepository
	txManager  transactionManager
}

func NewService(repository tripRepository, txManager transactionManager) *Service {
	return &Service{repository: repository, txManager: txManager}
}

func (service *Service) Create(
	ctx context.Context,
	input domain.CreateInput,
) (domain.Trip, error) {
	if err := input.Validate(); err != nil {
		return domain.Trip{}, err
	}
	trip := domain.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domain.StatusActive,
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
			domain.StatusActive,
			nil,
		); err != nil {
			return fmt.Errorf("add initial status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return domain.Trip{}, err
	}

	return trip, nil
}
