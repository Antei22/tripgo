package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test
type tripRepository interface {
	Get(context.Context, uuid.UUID) (domain.Trip, error)
}
