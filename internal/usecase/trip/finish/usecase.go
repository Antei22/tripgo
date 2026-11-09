package finish

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

func (service *Service) Finish(
	ctx context.Context,
	tripID uuid.UUID,
) (domain.Trip, error) {
	var result domain.Trip

	err := service.txManager.Do(ctx, func(txCtx context.Context) error {
		trip, err := service.repository.GetForUpdate(txCtx, tripID)
		if err != nil {
			return fmt.Errorf("lock trip: %w", err)
		}
		if err := trip.Finish(time.Now().UTC()); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}
		if err := service.repository.SaveFinished(txCtx, trip); err != nil {
			return fmt.Errorf("save finished trip: %w", err)
		}

		fromStatus := domain.StatusActive
		if err := service.repository.AddStatusHistory(
			txCtx,
			tripID,
			&fromStatus,
			domain.StatusCompleted,
			nil,
		); err != nil {
			return fmt.Errorf("add completed status history: %w", err)
		}

		result = trip
		return nil
	})
	if err != nil {
		return domain.Trip{}, err
	}

	return result, nil
}
