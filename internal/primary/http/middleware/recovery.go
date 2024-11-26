package middleware

import (
	// "fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for broken pipe
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				// Get stack trace
				stack := string(debug.Stack())
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				requestID := GetRequestID(c)

				if brokenPipe {
					logger.Error("Broken pipe",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("request_id", requestID),
					)
					c.Error(err.(error))
					c.Abort()
					return
				}

				logger.Error("Recovery from panic",
					zap.Any("error", err),
					zap.String("request", string(httpRequest)),
					zap.String("stack", stack),
					zap.String("request_id", requestID),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}