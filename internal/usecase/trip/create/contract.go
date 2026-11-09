package create

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
type tripRepository interface {
	Create(context.Context, domain.Trip) error
	AddStatusHistory(context.Context, uuid.UUID, *domain.Status, domain.Status, *string) error
}
type transactionManager interface {
	Do(context.Context, func(context.Context) error) error
}
