package database

import (
	"context"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type (
	queryKey  struct{}
	queryInfo struct {
		started               time.Time
		operation, collection string
	}
)

type queryTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func newQueryTracer() (*queryTracer, error) {
	histogram, err := otel.Meter("trip-service/postgres").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &queryTracer{Tracer: otelpgx.NewTracer(otelpgx.WithMeterProvider(noop.NewMeterProvider())), duration: histogram}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	info := queryInfo{started: time.Now(), operation: otelpgx.SQLOperationName(ctx, data.SQL)}
	sql := strings.ToLower(data.SQL)
	switch {
	case strings.Contains(sql, "trip_status_history"):
		info.collection = "trip_status_history"
	case strings.Contains(sql, "trips"):
		info.collection = "trips"
	}
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	if info.collection != "" {
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.collection.name", info.collection))
	}
	return context.WithValue(ctx, queryKey{}, info)
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	info, ok := ctx.Value(queryKey{}).(queryInfo)
	if ok {
		attrs := []attribute.KeyValue{attribute.String("db.operation.name", info.operation)}
		if info.collection != "" {
			attrs = append(attrs, attribute.String("db.collection.name", info.collection))
		}
		if data.Err != nil {
			attrs = append(attrs, attribute.String("error.type", "database_error"))
			trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", "database_error"))
		}
		t.duration.Record(ctx, time.Since(info.started).Seconds(), metric.WithAttributes(attrs...))
	}
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}
