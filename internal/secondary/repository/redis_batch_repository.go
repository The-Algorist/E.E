package repository

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
    
    "E.E/internal/core/domain"
    "E.E/internal/core/ports"
)

type RedisBatchRepository struct {
    *RedisBase
}

func NewRedisBatchRepository(config RedisConfig, logger *zap.Logger) (ports.BatchRepository, error) {
    base, err := newRedisBase(config, logger)
    if err != nil {
        return nil, err
    }
    return &RedisBatchRepository{RedisBase: base}, nil
}

func (r *RedisBatchRepository) StoreBatchResult(ctx context.Context, result *domain.BatchResult) error {
    key := fmt.Sprintf("batch:%s", result.BatchID)
    data, err := json.Marshal(result)
    if err != nil {
        return fmt.Errorf("failed to marshal batch result: %w", err)
    }

    // Use JobTTL from config
    if err := r.client.Set(ctx, key, data, r.config.JobTTL).Err(); err != nil {
        return fmt.Errorf("failed to store batch result: %w", err)
    }

    return nil
}

func (r *RedisBatchRepository) GetBatchResult(ctx context.Context, batchID string) (*domain.BatchResult, error) {
    key := fmt.Sprintf("batch:%s", batchID)
    data, err := r.client.Get(ctx, key).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, fmt.Errorf("batch result not found: %s", batchID)
        }
        return nil, fmt.Errorf("failed to get batch result: %w", err)
    }

    var result domain.BatchResult
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal batch result: %w", err)
    }

    return &result, nil
}

func (r *RedisBatchRepository) ListBatchResults(ctx context.Context, filter domain.BatchFilter) ([]*domain.BatchResult, error) {
    // Get all batch keys
    pattern := "batch:*"
    keys, err := r.client.Keys(ctx, pattern).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to list batch keys: %w", err)
    }

    var results []*domain.BatchResult
    for _, key := range keys {
        data, err := r.client.Get(ctx, key).Bytes()
        if err != nil {
            r.logger.Error("Failed to get batch result",
                zap.String("key", key),
                zap.Error(err))
            continue
        }

        var result domain.BatchResult
        if err := json.Unmarshal(data, &result); err != nil {
            r.logger.Error("Failed to unmarshal batch result",
                zap.String("key", key),
                zap.Error(err))
            continue
        }

        if matchesBatchFilter(&result, filter) {
            results = append(results, &result)
        }
    }

    return results, nil
}

func matchesBatchFilter(result *domain.BatchResult, filter domain.BatchFilter) bool {
    // If no filter is specified, include all results
    if filter.Status == "" && len(filter.JobIDs) == 0 {
        return true
    }

    // Check status if specified
    if filter.Status != "" {
        // Use Summary to determine status
        successRate := float64(result.Summary.SuccessCount) / float64(result.Summary.TotalJobs)
        currentStatus := "failed"
        if successRate == 1.0 {
            currentStatus = "success"
        } else if successRate > 0 {
            currentStatus = "partial"
        }
        
        if currentStatus != filter.Status {
            return false
        }
    }

    // Check job IDs if specified
    if len(filter.JobIDs) > 0 {
        jobMap := make(map[string]bool)
        // Add successful jobs
        for _, jobID := range result.Successful {
            jobMap[jobID] = true
        }
        // Add failed jobs
        for _, jobError := range result.Failed {
            jobMap[jobError.JobID] = true
        }
        
        // Check if all filtered job IDs are present
        for _, id := range filter.JobIDs {
            if !jobMap[id] {
                return false
            }
        }
    }

    return true
}

func (r *RedisBatchRepository) Close() error {
    return r.client.Close()
}