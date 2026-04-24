package routers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mw "mesh-crawler/core/middleware"
	"mesh-crawler/core/observability"
	handler "mesh-crawler/handler/crawler"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel" // used to pass the global propagator into otelecho

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Api() *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// ---- Basic safety middleware (can run early)
	e.Use(middleware.Recover())   // never panic the process on handler bugs
	e.Use(middleware.RequestID()) // adds X-Request-ID when missing

	// ---- Observability: OTel Resource, Providers, global propagator, /metrics handler, common instruments
	// Init() also sets the global W3C propagator (TraceContext + Baggage) via otel.SetTextMapPropagator(...)
	ctx := context.Background()
	obs := observability.Init(ctx)

	// ---- Tracing middleware MUST be registered before any logging/handlers.
	// It extracts the incoming traceparent/tracestate, creates the HTTP server span,
	// and injects the span into request.Context() so your handlers start as children.
	e.Use(otelecho.Middleware(
		"mesh-crawler",
		otelecho.WithPropagators(otel.GetTextMapPropagator()), // use the global one we set in Init()
	))

	// ---- Build a Zap logger and tee it into OpenTelemetry logs.
	// 'otelCore' mirrors every log record into the OTel Logs pipeline.
	zl, _ := zap.NewProduction()
	otelCore := otelzap.NewCore("mesh-crawler",
		otelzap.WithLoggerProvider(obs.LoggerProvider),
	)
	appLogger := zap.New(zapcore.NewTee(
		zl.Core(), // stdout/stderr JSON
		otelCore,  // OTel Logs export (correlates if a zap.Any("context", ctx) field is present)
	))

	// ---- Request logging middleware (AFTER tracing so the ctx has the span).
	// Your ZapRequestLogger should log with the request context when correlating logs to traces,
	// e.g. logger.With(zap.Any("context", c.Request().Context())).Info(...)
	e.Use(mw.ZapRequestLogger(appLogger))

	// ---- Optional basic Prometheus metrics middleware (Echo-specific)
	e.Use(echoprometheus.NewMiddleware("mesh_crawler"))

	// ---- Health & metrics endpoints
	e.GET("/health", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	e.GET("/metrics", echo.WrapHandler(obs.MetricsHandler)) // Prometheus /metrics (OTel exporter registered to a custom registry)

	// ---- Wire handler module with the app logger, OTel providers, and a tracer
	mod, _ := handler.NewModule(
		appLogger,
		obs.LoggerProvider,
		obs.MeterProvider,
		obs.TracerProvider.Tracer("mesh-crawler/handlers"),
	)

	// ---- Application routes
	e.GET("/", mod.Identify)
	e.GET("/identify", mod.Identify)
	e.GET("/wait", mod.Wait)
	e.POST("/crawl", mod.Crawl) // crawl is POST (can accept YAML or JSON; forwarded as JSON)
	e.GET("/load-test", mod.LoadTest)
	e.GET("/entities", mod.Entities)

	// ---- Start server with graceful shutdown on SIGINT/SIGTERM
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)

	// Start server in a goroutine so we can catch signals and shut down gracefully.
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("echo server error: %v", err)
		}
	}()

	// Wait for termination signal, then attempt a graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully stop Echo, flush Zap, and shutdown OTel providers.
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	_ = appLogger.Sync()
	obs.Shutdown(shutdownCtx)

	return e
}
