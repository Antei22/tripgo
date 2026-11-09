package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		record.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(a)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

func Setup(ctx context.Context, level slog.Level) (*slog.Logger, func(context.Context) error, error) {
	logger := slog.New(contextHandler{slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})}).With("service", "trip-service")
	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", "trip-service"), attribute.String("service.namespace", "tripgo"), attribute.String("service.version", "dev"), attribute.String("deployment.environment", "local")), resource.WithFromEnv(), resource.WithTelemetrySDK())
	if err != nil {
		return nil, nil, err
	}
	var te sdktrace.SpanExporter
	var me sdkmetric.Exporter
	switch protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol {
	case "", "http/protobuf":
		te, err = otlptracehttp.New(ctx, otlptracehttp.WithTimeout(2*time.Second))
		if err == nil {
			me, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithTimeout(2*time.Second))
		}
	case "grpc":
		te, err = otlptracegrpc.New(ctx, otlptracegrpc.WithTimeout(2*time.Second))
		if err == nil {
			me, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithTimeout(2*time.Second))
		}
	default:
		return nil, nil, fmt.Errorf("unsupported OTLP protocol %q", protocol)
	}
	if err != nil {
		if te != nil {
			_ = te.Shutdown(ctx)
		}
		return nil, nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(te))

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(me)))
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return logger, func(ctx context.Context) error { return errors.Join(mp.Shutdown(ctx), tp.Shutdown(ctx)) }, nil
}
