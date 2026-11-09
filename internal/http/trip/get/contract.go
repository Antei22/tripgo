package get

import (
	"context"

	"github.com/google/uuid"

	trips "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test
type usecase interface {
	Get(context.Context, uuid.UUID) (trips.Trip, error)
}
