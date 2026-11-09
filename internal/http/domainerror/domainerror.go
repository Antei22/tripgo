package domainerror

import (
	"errors"
	"net/http"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	api "github.com/antei22/tripgo/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status, code, title, detail := http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred"
	var validationErr *domain.ValidationError
	switch {
	case errors.As(err, &validationErr):
		status, code, title, detail = http.StatusBadRequest, "invalid_request", "Invalid request", validationErr.Error()
	case errors.Is(err, domain.ErrTripNotFound):
		status, code, title, detail = http.StatusNotFound, "trip_not_found", "Trip not found", "Trip with the specified ID was not found"
	case errors.Is(err, domain.ErrTripCompleted):
		status, code, title, detail = http.StatusConflict, "trip_completed", "Trip completed", "Trip has already been completed"
	case errors.Is(err, domain.ErrDriverBusy):
		status, code, title, detail = http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip"
	}
	uri := "https://tripgo.example/problems/"
	switch code {
	case "invalid_request":
		uri += "invalid-request"
	case "trip_not_found":
		uri += "trip-not-found"
	case "trip_completed":
		uri += "trip-completed"
	case "driver_busy":
		uri += "driver-busy"
	default:
		uri += "internal-error"
	}
	return status, api.Problem{Type: uri, Title: title, Status: int32(status), Detail: &detail, Code: code}
}
