package create

import (
	"context"

	trips "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
type usecase interface {
	Create(context.Context, trips.CreateInput) (trips.Trip, error)
}
