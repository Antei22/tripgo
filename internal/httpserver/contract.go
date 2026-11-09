package httpserver

import (
	"context"

	"github.com/google/uuid"

	trips "github.com/antei22/tripgo/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=httpserver_test
type createService interface {
	Create(context.Context, trips.CreateInput) (trips.Trip, error)
}
type getService interface {
	Get(context.Context, uuid.UUID) (trips.Trip, error)
}
type finishService interface {
	Finish(context.Context, uuid.UUID) (trips.Trip, error)
}
type pinger interface{ Ping(context.Context) error }
