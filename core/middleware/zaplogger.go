// core/middleware/zaplogger.go
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func ZapRequestLogger(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			req := c.Request()
			res := c.Response()
			ctx := req.Context() // <-- carry this

			fields := []zap.Field{
				zap.Any("context", ctx), // <-- CRITICAL for trace/span IDs
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", res.Status),
				zap.Duration("latency", latency),
				zap.String("remote_ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
				zap.Int64("bytes_in", req.ContentLength),
				zap.Int64("bytes_out", res.Size),
			}

			if err != nil {
				logger.Error("request failed", append(fields, zap.Error(err))...)
				return err
			}
			logger.Info("request completed", fields...)
			return nil
		}
	}
}
