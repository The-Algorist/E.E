package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"E.E/internal/primary/http/middleware"  // Import middleware from correct package
)

type Server struct {
	router *gin.Engine
	logger *zap.Logger
	srv    *http.Server
}

func NewServer(logger *zap.Logger) *Server {
	router := gin.New()

	// Add base middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	return &Server{
		router: router,
		logger: logger,
	}
}

func (s *Server) Start(port int) error {
	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server", zap.Int("port", port))
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	return s.srv.Shutdown(ctx)
}

func (s *Server) Router() *gin.Engine {
	return s.router
}