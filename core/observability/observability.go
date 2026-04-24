package observability

import (
	"context"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation" // <— we’ll set the global propagator
	logsdk "go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type Setup struct {
	Resource       *sdkresource.Resource
	LoggerProvider *logsdk.LoggerProvider
	TracerProvider *tracesdk.TracerProvider
	MeterProvider  *metricsdk.MeterProvider
	MetricsHandler http.Handler

	// Common instruments that handlers may reuse
	ReqCounter metric.Int64Counter
	LatencyMs  metric.Float64Histogram
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Init initializes OTEL Resource, Logs (OTLP/HTTP), Traces (OTLP/HTTP),
// Metrics via Prometheus, and sets a global W3C propagator so we
// EXTRACT incoming traceparent/tracestate on the server and INJECT them on clients.
func Init(ctx context.Context) *Setup {
	// ----- Resource: who is emitting telemetry?
	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			attribute.String("service.name", getenv("SERVICE_NAME", "mesh-crawler")),
			attribute.String("service.version", getenv("SERVICE_VERSION", "0.1.0")),
			attribute.String("deployment.environment", getenv("ENV", "dev")),
		),
	)
	Must(err)

	// ----- Propagation: make W3C TraceContext + Baggage the global default.
	// This is critical for distributed tracing across services (HTTP headers traceparent/tracestate).
	// otelecho middleware (server) and our HTTP client injections (client) will use this automatically.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// ----- Logs: OTLP/HTTP exporter + LoggerProvider
	logExp, err := otlploghttp.New(ctx,
		otlploghttp.WithInsecure(),
		otlploghttp.WithEndpoint(getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")),
	)
	Must(err)
	lp := logsdk.NewLoggerProvider(
		logsdk.WithProcessor(logsdk.NewBatchProcessor(logExp)),
		logsdk.WithResource(res),
	)

	// ----- Traces: OTLP/HTTP exporter + TracerProvider
	traceExp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")),
	)
	Must(err)
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(traceExp),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// ----- Metrics: Prometheus exporter + registry + /metrics handler
	reg := prometheus.NewRegistry()
	exp, err := otelprom.New(otelprom.WithRegisterer(reg))
	Must(err)
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithReader(exp),
		metricsdk.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	// ----- Common instruments to reuse in handlers
	meter := mp.Meter("mesh-crawler")
	reqCounter, _ := meter.Int64Counter("http_requests_total",
		metric.WithDescription("Total number of HTTP requests"))
	latency, _ := meter.Float64Histogram("http_request_duration_ms",
		metric.WithDescription("HTTP request duration in ms"))

	return &Setup{
		Resource:       res,
		LoggerProvider: lp,
		TracerProvider: tp,
		MeterProvider:  mp,
		MetricsHandler: h,
		ReqCounter:     reqCounter,
		LatencyMs:      latency,
	}
}

// Shutdown flushes providers; callers may ignore returned errors during shutdown.
func (s *Setup) Shutdown(ctx context.Context) {
	if s == nil {
		return
	}
	if s.LoggerProvider != nil {
		_ = s.LoggerProvider.Shutdown(ctx)
	}
	if s.TracerProvider != nil {
		_ = s.TracerProvider.Shutdown(ctx)
	}
	// MeterProvider has no Shutdown; readers/exporters terminate on process exit.
}
