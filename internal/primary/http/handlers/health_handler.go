package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HealthCheck func(context.Context) error

type HealthHandler struct {
	startTime time.Time
	checks    map[string]HealthCheck
	logger    *zap.Logger
}

func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		checks:    make(map[string]HealthCheck),
		logger:    logger,
	}
}

func (h *HealthHandler) AddCheck(name string, check HealthCheck) {
	h.checks[name] = check
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx := c.Request.Context()
	status := "ok"
	checks := make(map[string]string)

	for name, check := range h.checks {
		if err := check(ctx); err != nil {
			status = "error"
			checks[name] = fmt.Sprintf("error: %v", err)
		} else {
			checks[name] = "ok"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     status,
		"time":       time.Now().Unix(),
		"uptime":     time.Since(h.startTime).String(),
		"checks":     checks,
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
	})
}