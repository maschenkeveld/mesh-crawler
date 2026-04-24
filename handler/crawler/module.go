package handler

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/metric"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Module interface {
	Crawl(echo.Context) error
	Identify(echo.Context) error
	Health(echo.Context) error
	Wait(echo.Context) error
	LoadTest(echo.Context) error
	Entities(echo.Context) error
}

type module struct {
	Logger         *zap.Logger
	LoggerProvider *logsdk.LoggerProvider
	Tracer         trace.Tracer
	Meter          metric.Meter
	ReqCounter     metric.Int64Counter
	Latency        metric.Float64Histogram
}

var _ Module = &module{}

// Accept *zap.Logger (since we build it by teeing zap.Core with otelzap.NewCore)
func NewModule(logger *zap.Logger, lp *logsdk.LoggerProvider, meterProvider *metricsdk.MeterProvider, tracer trace.Tracer) (Module, error) {
	m := meterProvider.Meter("mesh-crawler/handler")

	req, _ := m.Int64Counter("http_requests_total")
	lat, _ := m.Float64Histogram("http_request_duration_ms")

	return &module{
		Logger:         logger,
		LoggerProvider: lp,
		Tracer:         tracer,
		Meter:          m,
		ReqCounter:     req,
		Latency:        lat,
	}, nil
}

// Entities implements the Module interface.
func (m *module) Entities(ctx echo.Context) error {
	return nil
}
