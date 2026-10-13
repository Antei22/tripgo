package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	api "github.com/antei22/tripgo/internal/generated"
	"github.com/antei22/tripgo/internal/trips"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ api.ServerInterface = (*Handler)(nil)

type Handler struct {
	service      *trips.Service
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	logger       *slog.Logger
}

func NewHandler(
	service *trips.Service,
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		service:      service,
		pool:         pool,
		queryTimeout: queryTimeout,
		logger:       logger,
	}
}

func (handler *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	if !hasJSONContentType(r) {
		handler.writeInvalidRequest(
			w,
			r,
			"Content-Type must be application/json",
		)
		return
	}

	var body api.CreateTripJSONRequestBody

	if err := decodeJSON(w, r, &body); err != nil {
		handler.writeInvalidRequest(
			w,
			r,
			"Request body contains invalid JSON",
		)
		return
	}

	if !validCreateTripBody(body) {
		handler.writeInvalidRequest(
			w,
			r,
			"Request body contains invalid trip data",
		)
		return
	}

	input := trips.CreateInput{
		UserID:   uuid.UUID(body.UserId),
		DriverID: uuid.UUID(body.DriverId),
		StartPoint: trips.Coordinates{
			Latitude:  body.StartPoint.Latitude,
			Longitude: body.StartPoint.Longitude,
		},
		EndPoint: trips.Coordinates{
			Latitude:  body.EndPoint.Latitude,
			Longitude: body.EndPoint.Longitude,
		},
		Price: body.Price,
	}

	trip, err := handler.service.Create(r.Context(), input)
	if err != nil {
		handler.writeServiceError(w, r, err)
		return
	}

	handler.logger.Info(
		"trip created",
		"trip_id",
		trip.ID.String(),
	)

	w.Header().Set(
		"Location",
		"/api/v1/trips/"+trip.ID.String(),
	)

	writeJSON(
		w,
		http.StatusCreated,
		tripToAPI(trip),
	)
}

func (handler *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := handler.service.Get(
		r.Context(),
		uuid.UUID(tripID),
	)
	if err != nil {
		handler.writeServiceError(w, r, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		tripToAPI(trip),
	)
}

func (handler *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := handler.service.Finish(
		r.Context(),
		uuid.UUID(tripID),
	)
	if err != nil {
		handler.writeServiceError(w, r, err)
		return
	}

	handler.logger.Info(
		"trip completed",
		"trip_id",
		trip.ID.String(),
	)

	writeJSON(
		w,
		http.StatusOK,
		tripToAPI(trip),
	)
}

func (handler *Handler) Health(
	w http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(
		w,
		http.StatusOK,
		api.HealthResponse{
			Status: api.Ok,
		},
	)
}

func (handler *Handler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		handler.queryTimeout,
	)
	defer cancel()

	if err := handler.pool.Ping(ctx); err != nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			api.HealthResponse{
				Status: api.Unavailable,
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		api.HealthResponse{
			Status: api.Ok,
		},
	)
}

func (handler *Handler) writeServiceError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, trips.ErrTripNotFound):
		writeProblem(
			w,
			r,
			http.StatusNotFound,
			"trip_not_found",
			"Trip not found",
			"Trip with the specified ID was not found",
		)

	case errors.Is(err, trips.ErrTripCompleted):
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"trip_completed",
			"Trip completed",
			"Trip has already been completed",
		)

	case errors.Is(err, trips.ErrDriverBusy):
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"driver_busy",
			"Driver busy",
			"Driver already has an active trip",
		)

	default:
		handler.logger.Error(
			"request failed",
			"method",
			r.Method,
			"path",
			r.URL.Path,
			"error",
			err,
		)

		writeProblem(
			w,
			r,
			http.StatusInternalServerError,
			"internal_error",
			"Internal server error",
			"An unexpected error occurred",
		)
	}
}

func (handler *Handler) writeInvalidRequest(
	w http.ResponseWriter,
	r *http.Request,
	detail string,
) {
	writeProblem(
		w,
		r,
		http.StatusBadRequest,
		"invalid_request",
		"Invalid request",
		detail,
	)
}

func (handler *Handler) writeParameterError(
	w http.ResponseWriter,
	r *http.Request,
	_ error,
) {
	handler.writeInvalidRequest(
		w,
		r,
		"Request parameters are invalid",
	)
}

func decodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		1<<20, // 1 MiB
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	// После первого объекта разрешён только конец файла.
	// Запрос вида "{} {}" должен быть отвергнут.
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New(
			"request body must contain a single JSON value",
		)
	}

	return err
}

func hasJSONContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "application/json"
}

func validCreateTripBody(
	body api.CreateTripJSONRequestBody,
) bool {
	if uuid.UUID(body.UserId) == uuid.Nil {
		return false
	}

	if uuid.UUID(body.DriverId) == uuid.Nil {
		return false
	}

	if body.Price < 0 {
		return false
	}

	if !validCoordinates(body.StartPoint) {
		return false
	}

	if !validCoordinates(body.EndPoint) {
		return false
	}

	return true
}

func validCoordinates(coordinates api.Coordinates) bool {
	if coordinates.Latitude < -90 ||
		coordinates.Latitude > 90 {
		return false
	}

	if coordinates.Longitude < -180 ||
		coordinates.Longitude > 180 {
		return false
	}

	return true
}

func tripToAPI(trip trips.Trip) api.Trip {
	return api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: trip.LastPositionAt,
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	title string,
	detail string,
) {
	instance := r.URL.Path

	problem := api.Problem{
		Type: "https://tripgo.example/problems/" +
			strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set(
		"Content-Type",
		"application/problem+json",
	)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(problem)
}
