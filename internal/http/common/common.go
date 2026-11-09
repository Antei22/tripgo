package common

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	trips "github.com/antei22/tripgo/internal/domain/trip"
	api "github.com/antei22/tripgo/internal/generated"
	"github.com/antei22/tripgo/internal/http/domainerror"
)

func WriteServiceError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	status, problem := domainerror.Problem(err)
	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", "error", err)
	}
	instance := r.URL.Path
	problem.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func WriteInvalidRequest(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", detail)
}

func DecodeJSON(
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

func HasJSONContentType(r *http.Request) bool {
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

func TripToAPI(trip trips.Trip) api.Trip {
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

func WriteJSON(
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

func WriteProblem(
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
		Status:   int32(status), //nolint:gosec // HTTP status codes fit in int32.
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
