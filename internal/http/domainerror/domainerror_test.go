package domainerror_test

import (
	"errors"
	"testing"

	domain "github.com/antei22/tripgo/internal/domain/trip"
	"github.com/antei22/tripgo/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		code   string
	}{{domain.ErrTripNotFound, 404, "trip_not_found"}, {domain.ErrTripCompleted, 409, "trip_completed"}, {domain.ErrDriverBusy, 409, "driver_busy"}, {errors.New("private"), 500, "internal_error"}} {
		status, body := domainerror.Problem(tt.err)
		if status != tt.status || body.Code != tt.code {
			t.Fatalf("%v: %d %+v", tt.err, status, body)
		}
	}
}
