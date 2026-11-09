package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	api "github.com/antei22/tripgo/internal/generated"
	"github.com/antei22/tripgo/internal/http/common"
	create "github.com/antei22/tripgo/internal/http/trip/create"
	finish "github.com/antei22/tripgo/internal/http/trip/finish"
	get "github.com/antei22/tripgo/internal/http/trip/get"
)

var _ api.ServerInterface = (*Handler)(nil)

type Handler struct {
	create       *create.Handler
	get          *get.Handler
	finish       *finish.Handler
	pool         pinger
	queryTimeout time.Duration
	logger       *slog.Logger
}

func NewHandler(c createService, g getService, f finishService, pool pinger, timeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{create: create.New(c, logger), get: get.New(g, logger), finish: finish.New(f, logger), pool: pool, queryTimeout: timeout, logger: logger}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, p api.CreateTripParams) {
	h.create.Handle(w, r, p)
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	h.get.Handle(w, r, id)
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	h.finish.Handle(w, r, id)
}

func (h *Handler) writeParameterError(w http.ResponseWriter, r *http.Request, _ error) {
	common.WriteInvalidRequest(w, r, "Request parameters are invalid")
}

func (handler *Handler) Health(
	w http.ResponseWriter,
	_ *http.Request,
) {
	common.WriteJSON(
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
		common.WriteJSON(
			w,
			http.StatusServiceUnavailable,
			api.HealthResponse{
				Status: api.Unavailable,
			},
		)
		return
	}

	common.WriteJSON(
		w,
		http.StatusOK,
		api.HealthResponse{
			Status: api.Ok,
		},
	)
}
