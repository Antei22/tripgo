package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/antei22/tripgo/internal/domain/trip"
)

type Service struct{ repository tripRepository }

func NewService(repository tripRepository) *Service { return &Service{repository: repository} }
func (service *Service) Get(
	ctx context.Context,
	tripID uuid.UUID,
) (domain.Trip, error) {
	return service.repository.Get(ctx, tripID)
}
