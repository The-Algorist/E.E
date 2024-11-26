package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	//"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"E.E/internal/primary/http"
	"E.E/internal/primary/http/handlers"
	"E.E/internal/core/services"
	"E.E/internal/secondary/repository"
	//"E.E/internal/secondary/s3"
	//"E.E/pkg/metrics"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize metrics
	// metricsClient := metrics.NewMetrics("encryption_service")

	// Create working directory
	workDir := "./tmp/storage"
	if err := os.MkdirAll(workDir, 0755); err != nil {
		logger.Fatal("Failed to create working directory", zap.Error(err))
	}

	// Initialize storage components
	// fileStorage, err := storage.NewFileStorage(workDir, logger)
	// if err != nil {
	// 	logger.Fatal("Failed to initialize file storage", zap.Error(err))
	// }

	// s3Client := s3.NewS3Client(logger)

	// Initialize Redis repositories
	redisConfig := repository.DefaultRedisConfig()
	if envURL := os.Getenv("REDIS_URL"); envURL != "" {
		redisConfig.URL = envURL
	}

	// Initialize job repository
	jobRepository, err := repository.NewRedisJobRepository(redisConfig, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis job repository", zap.Error(err))
	}
	defer jobRepository.Close()

	// Initialize batch repository
	batchRepository, err := repository.NewRedisBatchRepository(redisConfig, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis batch repository", zap.Error(err))
	}
	defer batchRepository.Close()

	// Initialize encryption service with both repositories
	encryptionService := services.NewEncryptionService(
		jobRepository,
		batchRepository,
		logger,
	)

	// Initialize batch service
	batchService := services.NewBatchService(
		encryptionService,
		jobRepository,
		batchRepository,
		logger,
	)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(logger)
	encryptionHandler := handlers.NewEncryptionHandler(
			encryptionService,
			logger,
	)
	batchHandler := handlers.NewBatchHandler(
		batchService,
		logger,
	)

	// Add Redis health check to the health handler
	healthHandler.AddCheck("redis", jobRepository.HealthCheck)

	// Initialize HTTP server
	server := http.NewServer(logger)

	// Setup router configuration
	routerConfig := http.RouterConfig{
		EncryptionHandler: encryptionHandler,
		BatchHandler:      batchHandler,
		HealthHandler:     healthHandler,
		Logger:           logger,
		RateLimit: struct {
			Enabled    bool
			Requests   int
			TimeWindow time.Duration
		}{
			Enabled:    true,
			Requests:   100,
			TimeWindow: time.Minute,
		},
	}

	// Setup routes
	http.SetupRouter(server.Router(), routerConfig)

	// Start server
	go func() {
		logger.Info("Starting server on :8080")
		if err := server.Start(8080); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}

