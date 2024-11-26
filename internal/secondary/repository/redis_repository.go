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

const (
    jobKeyPrefix = "job:"
)

type RedisJobRepository struct {
    *RedisBase
}

func NewRedisJobRepository(config RedisConfig, logger *zap.Logger) (ports.JobRepository, error) {
    base, err := newRedisBase(config, logger)
    if err != nil {
        return nil, err
    }
    return &RedisJobRepository{RedisBase: base}, nil
}

func (r *RedisJobRepository) Create(ctx context.Context, job *domain.EncryptionJob) error {
    data, err := json.Marshal(job)
    if err != nil {
        return fmt.Errorf("failed to marshal job: %w", err)
    }

    key := fmt.Sprintf("%s%s", jobKeyPrefix, job.ID)
    if err := r.RedisBase.client.Set(ctx, key, data, r.RedisBase.config.JobTTL).Err(); err != nil {
        return fmt.Errorf("failed to save job to Redis: %w", err)
    }

    return nil
}

func (r *RedisJobRepository) Update(ctx context.Context, job *domain.EncryptionJob) error {
    return r.Create(ctx, job) // Same operation for Redis
}

func (r *RedisJobRepository) Get(ctx context.Context, jobID string) (*domain.EncryptionJob, error) {
    key := fmt.Sprintf("%s%s", jobKeyPrefix, jobID)
    data, err := r.RedisBase.client.Get(ctx, key).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, nil // Job not found
        }
        return nil, fmt.Errorf("failed to get job from Redis: %w", err)
    }

    var job domain.EncryptionJob
    if err := json.Unmarshal(data, &job); err != nil {
        return nil, fmt.Errorf("failed to unmarshal job: %w", err)
    }

    return &job, nil
}

func (r *RedisJobRepository) Delete(ctx context.Context, jobID string) error {
    key := fmt.Sprintf("%s%s", jobKeyPrefix, jobID)
    if err := r.RedisBase.client.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("failed to delete job from Redis: %w", err)
    }

    return nil
}

func (r *RedisJobRepository) List(ctx context.Context) ([]*domain.EncryptionJob, error) {
    keys, err := r.RedisBase.client.Keys(ctx, jobKeyPrefix+"*").Result()
    if err != nil {
        return nil, fmt.Errorf("failed to list jobs from Redis: %w", err)
    }

    jobs := make([]*domain.EncryptionJob, 0, len(keys))
    for _, key := range keys {
        data, err := r.RedisBase.client.Get(ctx, key).Bytes()
        if err != nil {
            r.RedisBase.logger.Error("Failed to get job data", 
                zap.String("key", key),
                zap.Error(err),
            )
            continue
        }

        var job domain.EncryptionJob
        if err := json.Unmarshal(data, &job); err != nil {
            r.RedisBase.logger.Error("Failed to unmarshal job data",
                zap.String("key", key),
                zap.Error(err),
            )
            continue
        }

        jobs = append(jobs, &job)
    }

    return jobs, nil
}

func (r *RedisJobRepository) AddJobHistory(ctx context.Context, jobID string, entry domain.JobHistoryEntry) error {
    key := fmt.Sprintf("job_history:%s", jobID)
    data, err := json.Marshal(entry)
    if err != nil {
        return fmt.Errorf("failed to marshal job history entry: %w", err)
    }

    if err := r.RedisBase.client.RPush(ctx, key, data).Err(); err != nil {
        return fmt.Errorf("failed to add job history entry: %w", err)
    }

    // Set expiration if not already set
    r.RedisBase.client.Expire(ctx, key, r.RedisBase.config.JobTTL)
    return nil
}

func (r *RedisJobRepository) GetJobHistory(ctx context.Context, jobID string) ([]domain.JobHistoryEntry, error) {
    key := fmt.Sprintf("job_history:%s", jobID)
    data, err := r.RedisBase.client.LRange(ctx, key, 0, -1).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get job history: %w", err)
    }

    entries := make([]domain.JobHistoryEntry, 0, len(data))
    for _, item := range data {
        var entry domain.JobHistoryEntry
        if err := json.Unmarshal([]byte(item), &entry); err != nil {
            r.RedisBase.logger.Error("Failed to unmarshal job history entry",
                zap.String("job_id", jobID),
                zap.Error(err))
            continue
        }
        entries = append(entries, entry)
    }

    return entries, nil
}