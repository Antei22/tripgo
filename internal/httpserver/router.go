package httpserver

import (
	"net/http"

	api "github.com/antei22/tripgo/internal/generated"
	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	router := chi.NewRouter()

	return api.HandlerWithOptions(
		handler,
		api.ChiServerOptions{
			BaseRouter: router,

			ErrorHandlerFunc: handler.writeParameterError,
		},
	)
}
