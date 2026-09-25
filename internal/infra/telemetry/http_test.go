package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHTTPMiddlewareLogsCorrelatedCompletion(t *testing.T) {
	runtime, output, spans := newTestRuntime(t)
	router := gin.New()
	middleware, err := runtime.HTTPMiddleware()
	if err != nil {
		t.Fatalf("create HTTP middleware: %v", err)
	}
	router.Use(middleware)
	router.GET("/api/auth/context", func(context *gin.Context) {
		SetRequestOperation(context.Request.Context(), "user.account.load", CompletionDetails{
			Success:  Completion{Event: "user.account.loaded", Message: "User account loaded"},
			Rejected: Completion{Event: "user.sign_in.required", Message: "User needs to sign in"},
			Failed:   Completion{Event: "user.account.load.failed", Message: "Could not load user account"},
		})
		AddRequestLogAttributes(context.Request.Context(), slog.String("user_id", "user-123"))
		context.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/auth/context?ignored=secret", nil)
	request.Header.Set(requestIDHeader, "request-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Header().Get(requestIDHeader) != "request-123" {
		t.Fatalf("response request ID = %q, want request-123", response.Header().Get(requestIDHeader))
	}
	entry := decodeOnlyLogEntry(t, output)
	assertLogField(t, entry, "event", "user.account.loaded")
	assertLogField(t, entry, "operation", "user.account.load")
	assertLogField(t, entry, "method", http.MethodGet)
	assertLogField(t, entry, "route", "/api/auth/context")
	assertLogField(t, entry, "request_id", "request-123")
	assertLogField(t, entry, "outcome", "success")
	assertLogField(t, entry, "msg", "User account loaded")
	assertLogField(t, entry, "user_id", "user-123")
	if entry["duration_ms"] == nil {
		t.Fatal("completion log has no duration_ms")
	}
	traceID, ok := entry["trace_id"].(string)
	if !ok || len(traceID) != 32 {
		t.Fatalf("completion trace_id = %#v, want 32-character string", entry["trace_id"])
	}
	if bytes.Contains(output.Bytes(), []byte("ignored")) || bytes.Contains(output.Bytes(), []byte("secret")) {
		t.Fatalf("completion log contains raw query data: %s", output.String())
	}

	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Name() != "user.account.load" {
		t.Fatalf("span name = %q, want user.account.load", ended[0].Name())
	}
	if attributeValue(ended[0].Attributes(), "http.route") != "/api/auth/context" {
		t.Fatalf("span route = %q, want /api/auth/context", attributeValue(ended[0].Attributes(), "http.route"))
	}
}

func TestHTTPMiddlewareSuppressesSuccessfulHealthAndRecordsReadinessFailure(t *testing.T) {
	runtime, output, spans := newTestRuntime(t)
	router := gin.New()
	middleware, err := runtime.HTTPMiddleware()
	if err != nil {
		t.Fatalf("create HTTP middleware: %v", err)
	}
	router.Use(middleware)
	ready := true
	router.GET("/health/ready", func(context *gin.Context) {
		if !ready {
			context.Status(http.StatusServiceUnavailable)
			return
		}
		context.Status(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if output.Len() != 0 {
		t.Fatalf("successful readiness log = %s, want none", output.String())
	}
	if len(spans.Ended()) != 0 {
		t.Fatalf("successful readiness spans = %d, want none", len(spans.Ended()))
	}

	ready = false
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	entry := decodeOnlyLogEntry(t, output)
	assertLogField(t, entry, "event", "health.readiness.failed")
	assertLogField(t, entry, "operation", "health.readiness.check")
	assertLogField(t, entry, "msg", "API is not ready")
	assertLogField(t, entry, "error_category", "dependency_unavailable")
	assertLogField(t, entry, "outcome", "error")
	if len(spans.Ended()) != 1 || spans.Ended()[0].Name() != "health.readiness.check" {
		t.Fatalf("readiness failure spans = %#v, want one health.readiness span", spans.Ended())
	}
}

func newTestRuntime(t *testing.T) (*Runtime, *bytes.Buffer, *tracetest.SpanRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(healthSuppressingSampler{
			delegate: sdktrace.ParentBased(sdktrace.AlwaysSample()),
		}),
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
	})
	meterProvider := sdkmetric.NewMeterProvider()
	t.Cleanup(func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown meter provider: %v", err)
		}
	})

	var output bytes.Buffer
	logger := slog.New(newCorrelationHandler(slog.NewJSONHandler(&output, nil)))
	return &Runtime{
		logger:         logger,
		meterProvider:  meterProvider,
		tracerProvider: tracerProvider,
		propagator:     propagation.TraceContext{},
	}, &output, spanRecorder
}

func decodeOnlyLogEntry(t *testing.T, output *bytes.Buffer) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	var entry map[string]any
	if err := decoder.Decode(&entry); err != nil {
		t.Fatalf("decode completion log: %v; output = %s", err, output.String())
	}
	var extra map[string]any
	if err := decoder.Decode(&extra); err == nil {
		t.Fatalf("found extra completion log: %#v", extra)
	}
	return entry
}

func assertLogField(t *testing.T, entry map[string]any, field string, want any) {
	t.Helper()
	if got := entry[field]; got != want {
		t.Fatalf("log field %s = %#v, want %#v", field, got, want)
	}
}

func attributeValue(attributes []attribute.KeyValue, key attribute.Key) string {
	for _, current := range attributes {
		if current.Key == key {
			return current.Value.AsString()
		}
	}
	return ""
}
