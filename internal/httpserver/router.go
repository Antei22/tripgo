package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/antei22/tripgo/internal/generated"
)

func NewRouter(handler *Handler) http.Handler {
	router := chi.NewRouter()

	api.HandlerWithOptions(
		handler,
		api.ChiServerOptions{
			BaseRouter: router,

			ErrorHandlerFunc: handler.writeParameterError,
		},
	)
	return instrument(router, handler.logger)
}
