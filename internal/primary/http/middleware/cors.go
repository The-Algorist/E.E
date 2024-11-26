package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS middleware with configurable options
func CORS(config ...CORSConfig) gin.HandlerFunc {
	cfg := DefaultCORSConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Set CORS headers
		header := c.Writer.Header()
		
		// Check allowed origins
		if cfg.AllowOrigins[0] == "*" {
			header.Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			for _, allowedOrigin := range cfg.AllowOrigins {
				if allowedOrigin == origin {
					header.Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if len(cfg.AllowMethods) > 0 {
				header.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ","))
			}
			if len(cfg.AllowHeaders) > 0 {
				header.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ","))
			}
			if len(cfg.ExposeHeaders) > 0 {
				header.Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ","))
			}
			if cfg.AllowCredentials {
				header.Set("Access-Control-Allow-Credentials", "true")
			}
			if cfg.MaxAge > 0 {
				header.Set("Access-Control-Max-Age", cfg.MaxAge.String())
			}
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}