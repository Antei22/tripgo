package finish

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
type tripRepository interface {
	GetForUpdate(context.Context, uuid.UUID) (domain.Trip, error)
	SaveFinished(context.Context, domain.Trip) error
	AddStatusHistory(context.Context, uuid.UUID, *domain.Status, domain.Status, *string) error
}
type transactionManager interface {
	Do(context.Context, func(context.Context) error) error
}
