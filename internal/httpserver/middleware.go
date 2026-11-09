package httpserver

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/antei22/tripgo/internal/http/common"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func instrument(router chi.Router, logger *slog.Logger) http.Handler {
	meter := otel.Meter("trip-service/http")
	active, _ := meter.Int64UpDownCounter("http.server.active_requests")
	created, _ := meter.Int64Counter("tripgo.trips.created")
	completed, _ := meter.Int64Counter("tripgo.trips.completed")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := chi.NewRouteContext()
		route := "unmatched"
		if router.Match(ctx, r.Method, r.URL.Path) {
			route = ctx.RoutePattern()
		}
		attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", route)}
		span := trace.SpanFromContext(r.Context())
		span.SetName(r.Method + " " + route)
		span.SetAttributes(attrs...)
		if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
			labeler.Add(attribute.String("http.route", route))
		}
		active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		defer active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", recovered, "stack", string(debug.Stack()))
					if sw.status == 0 {
						common.WriteProblem(sw, r, http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred")
					}
				}
			}()
			router.ServeHTTP(sw, r)
		}()
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
			span.SetAttributes(attribute.String("error.type", http.StatusText(status)))
		}
		if status == http.StatusCreated && r.Method == http.MethodPost && route == "/api/v1/trips" {
			created.Add(r.Context(), 1)
		}
		if status == http.StatusOK && r.Method == http.MethodPost && route == "/api/v1/trips/{tripId}/finish" {
			completed.Add(r.Context(), 1)
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		logger.Log(r.Context(), level, "request handled", "method", r.Method, "route", route, "status", status, "duration_ms", time.Since(started).Milliseconds())
	})
	traced := otelhttp.NewHandler(inner, "HTTP request")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				logger.ErrorContext(r.Context(), "panic recovered", "panic", v, "stack", string(debug.Stack()))
				common.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred")
			}
		}()
		traced.ServeHTTP(w, r)
	})
}
