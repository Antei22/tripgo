package create

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	trips "github.com/antei22/tripgo/internal/domain/trip"
	api "github.com/antei22/tripgo/internal/generated"
	"github.com/antei22/tripgo/internal/http/common"
)

type Handler struct {
	service usecase
	logger  *slog.Logger
}

func New(service usecase, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (handler *Handler) Handle(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	if !common.HasJSONContentType(r) {
		common.WriteInvalidRequest(
			w,
			r,
			"Content-Type must be application/json",
		)
		return
	}

	var body request
	if err := common.DecodeJSON(w, r, &body); err != nil || !body.complete() {
		common.WriteInvalidRequest(w, r, "Request body contains invalid trip data")
		return
	}
	input := trips.CreateInput{UserID: *body.UserID, DriverID: *body.DriverID, Price: *body.Price, StartPoint: trips.Coordinates{Latitude: *body.StartPoint.Latitude, Longitude: *body.StartPoint.Longitude}, EndPoint: trips.Coordinates{Latitude: *body.EndPoint.Latitude, Longitude: *body.EndPoint.Longitude}}
	if err := input.Validate(); err != nil {
		common.WriteServiceError(w, r, err, handler.logger)
		return
	}

	trip, err := handler.service.Create(r.Context(), input)
	if err != nil {
		common.WriteServiceError(w, r, err, handler.logger)
		return
	}

	handler.logger.InfoContext(
		r.Context(),
		"trip created",
		"trip_id",
		trip.ID.String(),
	)

	w.Header().Set(
		"Location",
		"/api/v1/trips/"+trip.ID.String(),
	)

	common.WriteJSON(
		w,
		http.StatusCreated,
		common.TripToAPI(trip),
	)
}

type coordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}
type request struct {
	UserID     *uuid.UUID   `json:"user_id"`
	DriverID   *uuid.UUID   `json:"driver_id"`
	StartPoint *coordinates `json:"start_point"`
	EndPoint   *coordinates `json:"end_point"`
	Price      *int64       `json:"price"`
}

func (r request) complete() bool {
	return r.UserID != nil && r.DriverID != nil && r.Price != nil && r.StartPoint != nil && r.StartPoint.Latitude != nil && r.StartPoint.Longitude != nil && r.EndPoint != nil && r.EndPoint.Latitude != nil && r.EndPoint.Longitude != nil
}
