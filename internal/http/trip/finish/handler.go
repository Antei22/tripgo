package finish

import (
	"log/slog"
	"net/http"

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
	tripID api.TripId,
) {
	trip, err := handler.service.Finish(
		r.Context(),
		tripID,
	)
	if err != nil {
		common.WriteServiceError(w, r, err, handler.logger)
		return
	}

	handler.logger.InfoContext(
		r.Context(),
		"trip completed",
		"trip_id",
		trip.ID.String(),
	)

	common.WriteJSON(
		w,
		http.StatusOK,
		common.TripToAPI(trip),
	)
}
