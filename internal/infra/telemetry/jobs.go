package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// JobObserver owns generic lifecycle telemetry; job code supplies only its
// stable operation, durable identity, attempt, and business action.
type JobObserver struct {
	operation string
	logger    *slog.Logger
	tracer    trace.Tracer
	count     metric.Int64Counter
	duration  metric.Float64Histogram
	active    metric.Int64UpDownCounter
}

func (r *Runtime) NewJobObserver(operation string) (*JobObserver, error) {
	meter := r.meterProvider.Meter(instrumentationName)
	count, err := meter.Int64Counter("worker.job.executions")
	if err != nil {
		return nil, fmt.Errorf("create job counter: %w", err)
	}
	duration, err := meter.Float64Histogram("worker.job.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create job duration: %w", err)
	}
	active, err := meter.Int64UpDownCounter("worker.job.active")
	if err != nil {
		return nil, fmt.Errorf("create active job metric: %w", err)
	}
	return &JobObserver{operation: operation, logger: r.logger, tracer: r.tracerProvider.Tracer(instrumentationName), count: count, duration: duration, active: active}, nil
}
func (o *JobObserver) Run(ctx context.Context, id string, attempt int, action func(context.Context) error) error {
	ctx, span := o.tracer.Start(ctx, o.operation, trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	started := time.Now()
	attrs := []attribute.KeyValue{attribute.String("operation", o.operation)}
	options := metric.WithAttributes(attrs...)
	o.active.Add(ctx, 1, options)
	defer o.active.Add(ctx, -1, options)
	logger := o.logger.With("operation", o.operation, "job_id", id, "attempt", attempt, "trace_id", span.SpanContext().TraceID().String())
	logger.InfoContext(ctx, "Worker job started", "event", o.operation+".started")
	err := action(ctx)
	outcome := "success"
	if err != nil {
		outcome = "failure"
		category := "processing_failed"
		if errors.Is(err, context.Canceled) {
			category = "canceled"
		} else if errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		span.SetStatus(codes.Error, category)
		logger.ErrorContext(ctx, "Worker job failed", "event", o.operation+".failed", "duration_ms", time.Since(started).Milliseconds(), "error_category", category)
	} else {
		logger.InfoContext(ctx, "Worker job completed", "event", o.operation+".completed", "duration_ms", time.Since(started).Milliseconds())
	}
	attrs = append(attrs, attribute.String("outcome", outcome))
	o.count.Add(ctx, 1, metric.WithAttributes(attrs...))
	o.duration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attrs...))
	return err
}
