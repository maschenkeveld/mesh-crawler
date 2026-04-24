package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"mesh-crawler/core/errors"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"

	// OpenTelemetry bits
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	// Zap for structured logs
	"go.uber.org/zap"
)

// We call peers over HTTP and hit their /crawl too.
const (
	protocol = "http://"
	path     = "/crawl"
)

// Crawl accepts **YAML or JSON** input, normalizes into our internal Payload,
// then forwards to upstreams **as JSON**. It emits OTel traces/metrics/logs and
// participates in distributed traces by extracting + injecting W3C headers.
func (m *module) Crawl(c echo.Context) error {
	// 1) Always start from the request context (Echo placed it on the request).
	ctx := c.Request().Context()

	// 2) DEFENSIVE extraction of inbound parent trace context.
	//    otelecho middleware already tries to do this, but adding it here ensures
	//    that—even if middleware order changes—we still join the caller's trace.
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request().Header))

	// 3) For latency metric.
	start := time.Now()

	// 4) Start a handler span (child of the extracted parent).
	//    We annotate it with standard HTTP attributes for easy querying.
	ctx, span := m.Tracer.Start(ctx, "handler.crawl",
		trace.WithAttributes(
			attribute.String("http.method", c.Request().Method),
			attribute.String("http.route", c.Path()),              // matched route
			attribute.String("http.target", c.Request().URL.Path), // raw path
		),
	)
	defer span.End()

	// 5) Build a logger *bound to this request context*.
	//    The otelzap bridge recognizes zap fields of type context.Context and will
	//    copy trace/span IDs into the exported OpenTelemetry log records.
	log := m.Logger.With(zap.Any("context", ctx))

	// 6) Prepare our response structure that we’ll fill incrementally.
	resp := &CrawlResponse{
		IncomingHeaders:   map[string]string{},
		StatusCode:        http.StatusOK,
		Reason:            "success",
		UpstreamResponses: nil,
	}

	// 7) Identity / environment info (used for observability + response).
	//    If any are missing we mark the span as error and return 500 with a reason.
	name, ok := os.LookupEnv("SERVICE_NAME")
	if !ok {
		resp.Reason = errors.ErrNoNameEnvSet.Error()
		resp.StatusCode = http.StatusInternalServerError
		span.RecordError(errors.ErrNoNameEnvSet)
		span.SetStatus(codes.Error, resp.Reason)
		return echo.NewHTTPError(resp.StatusCode, resp)
	}
	version, ok := os.LookupEnv("SERVICE_VERSION")
	if !ok {
		resp.Reason = errors.ErrNoVersionEnvSet.Error()
		resp.StatusCode = http.StatusInternalServerError
		span.RecordError(errors.ErrNoVersionEnvSet)
		span.SetStatus(codes.Error, resp.Reason)
		log.Error("missing SERVICE_VERSION env")
		return echo.NewHTTPError(resp.StatusCode, resp)
	}
	hostname, ok := os.LookupEnv("HOSTNAME")
	if !ok {
		resp.Reason = errors.ErrNoHostnameEnvSet.Error()
		resp.StatusCode = http.StatusInternalServerError
		span.RecordError(errors.ErrNoHostnameEnvSet)
		span.SetStatus(codes.Error, resp.Reason)
		return echo.NewHTTPError(resp.StatusCode, resp)
	}
	zone, ok := os.LookupEnv("MESH_ZONE")
	if !ok {
		resp.Reason = errors.ErrNoZoneEnvSet.Error()
		resp.StatusCode = http.StatusInternalServerError
		span.RecordError(errors.ErrNoZoneEnvSet)
		span.SetStatus(codes.Error, resp.Reason)
		return echo.NewHTTPError(resp.StatusCode, resp)
	}

	resp.Name = name
	resp.Version = version
	resp.Hostname = hostname
	resp.Zone = zone
	resp.FullPath = c.Request().URL.Path

	// 8) Copy *all* incoming headers into the response for visibility.
	for k, vs := range c.Request().Header {
		for _, v := range vs {
			resp.IncomingHeaders[k] = v
		}
	}

	// 9) Parse inbound body as YAML or JSON.
	//    - We normalize Content-Type (strip params, use first of comma-separated).
	//    - YAML gets unmarshaled to the same Go struct; from here on we’re format-agnostic.
	payload, inFmt, err := parsePayloadYAMLorJSON(c)
	if err != nil {
		resp.Reason = err.Error()
		resp.StatusCode = http.StatusBadRequest
		span.RecordError(err)
		span.SetStatus(codes.Error, resp.Reason)
		return echo.NewHTTPError(resp.StatusCode, resp)
	}

	// 10) Record what came in (great for dashboards: YAML vs JSON usage).
	span.SetAttributes(attribute.String("request.payload_format", inFmt))
	log.Info("crawl: parsed inbound payload", zap.String("format", inFmt))

	// 11) For each “upstream” hop in the payload, call it as JSON and collect its response.
	for _, up := range payload.Upstreams {
		upResp := &CrawlResponse{}

		// Forward only the *rest* of the chain to the next hop.
		next := Payload{Upstreams: up.Upstreams}

		// Call the upstream. Errors are captured inside and returned;
		// we collect them per-hop but do not fail the entire handler.
		if err := m.evaluateNextResponse(ctx, log, name, c.Request().Header, next, up.Host, upResp); err != nil {
			upResp.Reason = err.Error()
		}
		resp.UpstreamResponses = append(resp.UpstreamResponses, upResp)
	}

	// 12) Emit metrics (counter + latency) with useful attributes.
	attrs := metric.WithAttributes(
		attribute.String("http.route", c.Path()),
		attribute.String("http.method", c.Request().Method),
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("request.payload_format", inFmt),
	)
	m.ReqCounter.Add(ctx, 1, attrs)
	latencyMs := float64(time.Since(start).Milliseconds())
	m.Latency.Record(ctx, latencyMs, attrs)

	// 13) Finalize span: set status + useful attributes for backend querying.
	if resp.StatusCode >= 500 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Float64("http.server.duration_ms", latencyMs),
	)

	// 14) Return the structured response.
	return c.JSON(resp.StatusCode, resp)
}

// parsePayloadYAMLorJSON reads the body and populates our internal Payload from
// either YAML or JSON. We normalize incoming Content-Type like:
//   - drop parameters (e.g. "; charset=utf-8")
//   - pick the first of comma-separated types (e.g. "application/json, text/yaml")
//
// Returns the parsed Payload and a string flag "yaml" or "json" for observability.
func parsePayloadYAMLorJSON(c echo.Context) (*Payload, string, error) {
	// Normalize content-type: remove params and pick first media type.
	ct := strings.ToLower(c.Request().Header.Get("Content-Type"))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if j := strings.Index(ct, ","); j >= 0 {
		ct = strings.TrimSpace(ct[:j])
	}

	var p Payload

	switch ct {
	case "text/yaml", "application/x-yaml", "application/yaml":
		// If YAML, read the raw body and unmarshal.
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, "", err
		}
		if err := yaml.Unmarshal(body, &p); err != nil {
			return nil, "", err
		}
		return &p, "yaml", nil

	// Treat everything else as JSON (including empty/unknown CT).
	default:
		// Echo's Bind uses JSON decoder for JSON Content-Type (and often even without it).
		if err := c.Bind(&p); err != nil {
			return nil, "", err
		}
		return &p, "json", nil
	}
}

// evaluateNextResponse performs ONE upstream hop and **always sends JSON**.
// It creates a child span, injects W3C trace headers so the upstream joins our trace,
// forwards safe headers (skips Content-Type/Length etc.), and fills upstreamResponse.
func (m *module) evaluateNextResponse(
	parentCtx context.Context, // parent ctx with active span
	log *zap.Logger, // logger that already carries the parent ctx
	name string, // our service name for a custom header
	headers http.Header, // incoming headers to selectively forward
	payload Payload, // the JSON payload we push upstream (we'll marshal it)
	host string, // upstream host (without scheme/path)
	upstreamResponse *CrawlResponse, // will be filled from upstream response
) error {
	// Ensure header map is non-nil so we can fill it safely.
	if upstreamResponse.IncomingHeaders == nil {
		upstreamResponse.IncomingHeaders = map[string]string{}
	}

	// Start a CHILD span that represents this upstream call.
	ctx, span := m.Tracer.Start(parentCtx, "crawl.upstream",
		trace.WithAttributes(
			attribute.String("net.peer.name", host),
			attribute.String("http.method", http.MethodPost),
			attribute.String("http.target", path),
			attribute.String("outbound.payload_format", "json"), // we always send JSON upstream
		),
	)
	defer span.End()

	// Add upstream host to logs to make stdout more helpful.
	log = log.With(zap.String("upstream.host", host))

	// Standardize outbound payload to JSON.
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		upstreamResponse.StatusCode = http.StatusInternalServerError
		upstreamResponse.Reason = fmt.Sprintf("failed to marshal payload: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, upstreamResponse.Reason)
		log.Error("marshal payload failed", zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s%s%s", protocol, host, path)

	// Create a request bound to the CHILD span's context.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		upstreamResponse.StatusCode = http.StatusInternalServerError
		upstreamResponse.Reason = fmt.Sprintf("failed to build request: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, upstreamResponse.Reason)
		log.Error("build request failed", zap.Error(err))
		return err
	}

	// Explicit headers we control (normalize to JSON outbound).
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mesh-crawler-requester", name)

	// Forward *safe* incoming headers.
	// Skip hop-by-hop headers and ones we explicitly set to avoid collisions like
	// "application/json,text/yaml" → 415 Unsupported Media Type.
	for key, values := range headers {
		lk := strings.ToLower(key)
		if lk == "connection" || lk == "keep-alive" || lk == "proxy-authenticate" ||
			lk == "proxy-authorization" || lk == "te" || lk == "trailers" ||
			lk == "transfer-encoding" || lk == "upgrade" || lk == "host" ||
			lk == "content-type" || lk == "content-length" {
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	// Inject W3C trace context so the upstream becomes part of the SAME trace.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Execute with a bounded timeout (tune as needed for your environment).
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		upstreamResponse.StatusCode = http.StatusServiceUnavailable
		upstreamResponse.Reason = fmt.Sprintf("upstream request failed: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, upstreamResponse.Reason)
		log.Error("upstream request failed", zap.Error(err))
		return err
	}
	defer res.Body.Close()

	// Record upstream status on both our response and child span.
	upstreamResponse.StatusCode = res.StatusCode
	span.SetAttributes(attribute.Int("http.status_code", res.StatusCode))

	// Capture upstream response headers (keep the last value if multiple).
	for k, vs := range res.Header {
		if len(vs) > 0 {
			upstreamResponse.IncomingHeaders[k] = vs[len(vs)-1]
		}
	}

	// Read upstream body for inclusion in our response (or to parse as JSON).
	body, err := io.ReadAll(res.Body)
	if err != nil {
		upstreamResponse.Reason = fmt.Sprintf("failed reading upstream body: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, upstreamResponse.Reason)
		log.Error("read upstream body failed", zap.Error(err))
		return err
	}

	bodyStr := strings.TrimSpace(string(body))
	ct := strings.ToLower(res.Header.Get("Content-Type"))
	isJSONCT := strings.Contains(ct, "application/json")
	looksJSON := len(bodyStr) > 0 && (bodyStr[0] == '{' || bodyStr[0] == '[')

	// If the upstream looks like JSON, parse it into our CrawlResponse. Otherwise,
	// keep the raw text in Reason and preserve the real status code.
	if (isJSONCT || looksJSON) && len(bodyStr) > 0 {
		if err := json.Unmarshal(body, upstreamResponse); err != nil {
			// Not JSON after all—keep raw text in Reason.
			if upstreamResponse.Reason == "" {
				upstreamResponse.Reason = bodyStr
			}
			log.Info("upstream non-json body", zap.String("reason", bodyStr))
			return nil
		}
		// Preserve HTTP status + headers even if upstream JSON contains different fields.
		upstreamResponse.StatusCode = res.StatusCode
		if upstreamResponse.IncomingHeaders == nil {
			upstreamResponse.IncomingHeaders = map[string]string{}
		}
		log.Info("upstream json parsed",
			zap.Int("status", res.StatusCode),
			zap.String("content_type", ct),
		)
		return nil
	}

	// Plain-text body: stash as text reason. If empty, fall back to status text.
	if bodyStr != "" {
		upstreamResponse.Reason = bodyStr
	} else {
		upstreamResponse.Reason = http.StatusText(res.StatusCode)
	}

	log.Info("upstream completed",
		zap.Int("status", res.StatusCode),
		zap.String("content_type", ct),
	)

	return nil
}
