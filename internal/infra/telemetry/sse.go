package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type SSEObserver struct {
	operation string
	active    metric.Int64UpDownCounter
	rejected  metric.Int64Counter
	failures  metric.Int64Counter
}

func (runtime *Runtime) NewSSEObserver(operation string) (*SSEObserver, error) {
	meter := runtime.meterProvider.Meter(instrumentationName)
	active, err := meter.Int64UpDownCounter(
		"consumel.sse.active_connections",
		metric.WithDescription("Number of active Consumel server-sent event connections."),
	)
	if err != nil {
		return nil, fmt.Errorf("create SSE active-connection metric: %w", err)
	}
	rejected, err := meter.Int64Counter(
		"consumel.sse.rejected_connections",
		metric.WithDescription("Number of rejected Consumel server-sent event connections."),
	)
	if err != nil {
		return nil, fmt.Errorf("create SSE rejected-connection metric: %w", err)
	}
	failures, err := meter.Int64Counter(
		"consumel.sse.delivery_failures",
		metric.WithDescription("Number of Consumel server-sent event delivery failures."),
	)
	if err != nil {
		return nil, fmt.Errorf("create SSE delivery-failure metric: %w", err)
	}
	return &SSEObserver{operation: operation, active: active, rejected: rejected, failures: failures}, nil
}

func (observer *SSEObserver) Connected(ctx context.Context) func() {
	options := metric.WithAttributes(attribute.String("operation", observer.operation))
	observer.active.Add(ctx, 1, options)
	return func() { observer.active.Add(context.Background(), -1, options) }
}

func (observer *SSEObserver) Rejected(ctx context.Context) {
	observer.rejected.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", observer.operation)))
}

func (observer *SSEObserver) DeliveryFailed(ctx context.Context) {
	observer.failures.Add(ctx, 1, metric.WithAttributes(attribute.String("operation", observer.operation)))
}
