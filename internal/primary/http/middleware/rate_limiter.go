package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   RateLimitConfig
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(rl.config.TimeWindow/time.Duration(rl.config.Requests)), rl.config.Requests)
		rl.limiters[key] = limiter
	}

	return limiter
}

// RateLimit middleware with configurable options
func RateLimit(config ...RateLimitConfig) gin.HandlerFunc {
	cfg := DefaultRateLimitConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	limiter := NewRateLimiter(cfg)

	return func(c *gin.Context) {
		key := cfg.KeyFunc(c)
		if !limiter.getLimiter(key).Allow() {
			c.JSON(429, gin.H{
				"error": "Too many requests",
				"retry_after": cfg.TimeWindow.Seconds(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}