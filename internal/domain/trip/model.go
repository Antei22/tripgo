package trip

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
	ErrDriverBusy    = errors.New("driver already has an active trip")
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

type CreateInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

// Finish enforces the trip state transition.
func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}

// ValidationError describes a field that violates a trip rule.
type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Reason }
func (in CreateInput) Validate() error {
	const outOfRange = "out of range"
	switch {
	case in.UserID == uuid.Nil:
		return &ValidationError{Field: "user_id", Reason: "must be a non-zero UUID"}
	case in.DriverID == uuid.Nil:
		return &ValidationError{Field: "driver_id", Reason: "must be a non-zero UUID"}
	case in.Price < 0:
		return &ValidationError{Field: "price", Reason: "must be non-negative"}
	case in.StartPoint.Latitude < -90 || in.StartPoint.Latitude > 90:
		return &ValidationError{Field: "start_point.latitude", Reason: outOfRange}
	case in.StartPoint.Longitude < -180 || in.StartPoint.Longitude > 180:
		return &ValidationError{Field: "start_point.longitude", Reason: outOfRange}
	case in.EndPoint.Latitude < -90 || in.EndPoint.Latitude > 90:
		return &ValidationError{Field: "end_point.latitude", Reason: outOfRange}
	case in.EndPoint.Longitude < -180 || in.EndPoint.Longitude > 180:
		return &ValidationError{Field: "end_point.longitude", Reason: outOfRange}
	}
	return nil
}
