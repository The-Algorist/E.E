package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger middleware with configurable options
func Logger(log *zap.Logger, config ...LogConfig) gin.HandlerFunc {
	var cfg LogConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		// Skip logging for specified paths
		if cfg.SkipPaths != nil {
			for _, path := range cfg.SkipPaths {
				if path == c.Request.URL.Path {
					c.Next()
					return
				}
			}
		}

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		requestID := GetRequestID(c)

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create custom response writer to capture response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add custom fields if configured
		if cfg.CustomFields != nil {
			customFields := cfg.CustomFields(c)
			for k, v := range customFields {
				fields = append(fields, zap.Any(k, v))
			}
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			log.Error("Server error", fields...)
		case c.Writer.Status() >= 400:
			log.Warn("Client error", fields...)
		default:
			log.Info("Request completed", fields...)
		}
	}
}