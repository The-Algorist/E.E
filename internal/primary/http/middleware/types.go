package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	
)

// Configuration types
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge          time.Duration
}

type RateLimitConfig struct {
	Requests   int
	TimeWindow time.Duration
	// Key function to identify clients (e.g., by IP, by API key)
	KeyFunc    func(c *gin.Context) string
}

type LogConfig struct {
	SkipPaths []string
	// Custom fields to add to logs
	CustomFields func(c *gin.Context) map[string]interface{}
}

// Default configurations
var (
	DefaultCORSConfig = CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:          12 * time.Hour,
	}

	DefaultRateLimitConfig = RateLimitConfig{
		Requests:   100,
		TimeWindow: time.Minute,
		KeyFunc:    func(c *gin.Context) string { return c.ClientIP() },
	}
)