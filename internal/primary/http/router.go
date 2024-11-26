package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"E.E/internal/primary/http/handlers"
	"E.E/internal/primary/http/middleware"
)


type RouterConfig struct {
	EncryptionHandler *handlers.EncryptionHandler
	BatchHandler      *handlers.BatchHandler
	HealthHandler     *handlers.HealthHandler
	Logger           *zap.Logger
	RateLimit        struct {
		Enabled    bool
		Requests   int
		TimeWindow time.Duration
	}
}

func SetupRouter(router *gin.Engine, cfg RouterConfig) {
	// API rate limiter if enabled
	var apiLimiter gin.HandlerFunc
	if cfg.RateLimit.Enabled {
		rateLimitConfig := middleware.RateLimitConfig{
			Requests:   cfg.RateLimit.Requests,
			TimeWindow: cfg.RateLimit.TimeWindow,
			KeyFunc:    func(c *gin.Context) string { return c.ClientIP() }, // Default to IP-based rate limiting
		}
		apiLimiter = middleware.RateLimit(rateLimitConfig)
	}

	// Health check endpoint (no rate limit)
	router.GET("/health", cfg.HealthHandler.Check)

	// Metrics endpoint (no rate limit)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := router.Group("/api/v1")
	if apiLimiter != nil {
		v1.Use(apiLimiter)
	}
	{
		// Encryption endpoints
		v1.POST("/encrypt", cfg.EncryptionHandler.StartEncryption)
		v1.GET("/status/:jobId", cfg.EncryptionHandler.GetStatus)
		v1.POST("/job/:jobId/pause", cfg.EncryptionHandler.PauseJob)
		v1.POST("/job/:jobId/resume", cfg.EncryptionHandler.ResumeJob)
		v1.POST("/job/:jobId/stop", cfg.EncryptionHandler.StopJob)
		v1.POST("/engine/stop", cfg.EncryptionHandler.StopEngine)
		v1.GET("/jobs", cfg.EncryptionHandler.ListJobs)
		v1.GET("/jobs/status", cfg.EncryptionHandler.JobsStatus)

		// Add batch endpoints
		v1.GET("/batch/:batchId", cfg.BatchHandler.GetBatchOperation)
		v1.GET("/batch", cfg.BatchHandler.ListBatchResults)
	}

	// Not found handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": "Route not found",
		})
	})
}