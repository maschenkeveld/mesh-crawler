package handler

import (
	"fmt"
	"io"
	"mesh-crawler/core/errors"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (m *module) Identify(c echo.Context) error {
	ctx := c.Request().Context()
	start := time.Now()

	// Start a span for this handler
	ctx, span := m.Tracer.Start(ctx, "handler.identify",
		trace.WithAttributes(
			attribute.String("http.method", c.Request().Method),
			attribute.String("http.route", c.Path()),
			attribute.String("http.target", c.Request().URL.Path),
		),
	)
	defer span.End()

	// Logger that always carries the request context (so OTel bridge attaches trace ids)
	log := m.Logger.With(zap.Any("context", ctx))

	// Helpful trace fields for stdout logs
	sc := span.SpanContext()
	traceFields := []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}

	log.Info("Incoming request",
		append(traceFields,
			zap.String("method", c.Request().Method),
			zap.String("url", c.Request().URL.Path),
		)...,
	)

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read body failed")
		log.Error("Error reading request body", append(traceFields, zap.Error(err))...)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read request body")
	}

	name, ok := os.LookupEnv("SERVICE_NAME")
	if !ok {
		err := errors.ErrNoNameEnvSet
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	version, ok := os.LookupEnv("SERVICE_VERSION")
	if !ok {
		err := errors.ErrNoVersionEnvSet
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	hostname, ok := os.LookupEnv("SERVICE_HOSTNAME")
	if !ok {
		err := errors.ErrNoNameEnvSet
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	zone, ok := os.LookupEnv("MESH_ZONE")
	if !ok {
		err := errors.ErrNoZoneEnvSet
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Hostname        string            `json:"hostname"`
		Zone            string            `json:"zone"`
		FullPath        string            `json:"fullPath"`
		IncomingHeaders map[string]string `json:"incomingHeaders"`
		Payload         string            `json:"payload"`
	}{
		Name: name, Version: version, Hostname: hostname, Zone: zone,
		FullPath: c.Request().URL.Path, IncomingHeaders: map[string]string{}, Payload: string(body),
	}

	for key, values := range c.Request().Header {
		for _, value := range values {
			resp.IncomingHeaders[key] = value
			log.Info("Request header", append(traceFields, zap.String("key", key), zap.String("value", value))...)
		}
	}

	log.Info("Request payload", append(traceFields, zap.String("payload", string(body)))...)
	log.Info("Response payload", append(traceFields, zap.String("payload", fmt.Sprintf("%+v", resp)))...)

	attrs := metric.WithAttributes(
		attribute.String("http.route", c.Path()),
		attribute.String("http.method", c.Request().Method),
	)
	m.ReqCounter.Add(ctx, 1, attrs)
	latencyMs := float64(time.Since(start).Milliseconds())
	m.Latency.Record(ctx, latencyMs, attrs)

	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	span.SetStatus(codes.Ok, "ok")

	return c.JSON(http.StatusOK, resp)
}
