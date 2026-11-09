package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	trip "github.com/antei22/tripgo/internal/domain/trip"
)

func TestFinish(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name    string
		status  trip.Status
		wantErr bool
	}{{"active", trip.StatusActive, false}, {"completed", trip.StatusCompleted, true}} {
		t.Run(tt.name, func(t *testing.T) {
			tr := trip.Trip{Status: tt.status}
			err := tr.Finish(at)
			if tt.wantErr {
				if !errors.Is(err, trip.ErrTripCompleted) {
					t.Fatalf("error = %v", err)
				}
				if tr.FinishedAt != nil {
					t.Fatal("completed trip changed")
				}
				return
			}
			if err != nil || tr.Status != trip.StatusCompleted || tr.FinishedAt == nil || !tr.FinishedAt.Equal(at) {
				t.Fatalf("trip = %+v, error = %v", tr, err)
			}
		})
	}
}

func TestCreateInputValidation(t *testing.T) {
	valid := trip.CreateInput{UserID: uuid.New(), DriverID: uuid.New()}
	for _, tt := range []struct {
		name   string
		change func(*trip.CreateInput)
		field  string
	}{
		{"valid", func(*trip.CreateInput) {}, ""},
		{"user", func(v *trip.CreateInput) { v.UserID = uuid.Nil }, "user_id"},
		{"driver", func(v *trip.CreateInput) { v.DriverID = uuid.Nil }, "driver_id"},
		{"price", func(v *trip.CreateInput) { v.Price = -1 }, "price"},
		{"latitude", func(v *trip.CreateInput) { v.StartPoint.Latitude = 91 }, "start_point.latitude"},
		{"longitude", func(v *trip.CreateInput) { v.EndPoint.Longitude = 181 }, "end_point.longitude"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.change(&input)
			err := input.Validate()
			if tt.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var ve *trip.ValidationError
			if !errors.As(err, &ve) || ve.Field != tt.field {
				t.Fatalf("got %v, want field %s", err, tt.field)
			}
		})
	}
}
