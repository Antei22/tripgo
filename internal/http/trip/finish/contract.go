package finish

import (
	"context"

	"github.com/google/uuid"

	trips "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
type usecase interface {
	Finish(context.Context, uuid.UUID) (trips.Trip, error)
}
