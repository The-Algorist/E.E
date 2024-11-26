package repository

import (
    "context"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
	"fmt"
)

type RedisBase struct {
    client *redis.Client
    logger *zap.Logger
    config RedisConfig
}

func newRedisBase(config RedisConfig, logger *zap.Logger) (*RedisBase, error) {
    opts := &redis.Options{
        Addr:         config.URL,
        Password:     config.Password,
        DB:          config.DB,
        MaxRetries:   config.MaxRetries,
        MinIdleConns: config.MinIdleConns,
        PoolSize:     config.PoolSize,
        PoolTimeout:  config.PoolTimeout,
        DialTimeout:  config.ConnectTimeout,
        ReadTimeout:  config.ReadTimeout,
        WriteTimeout: config.WriteTimeout,
    }

    client := redis.NewClient(opts)

    ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
    defer cancel()
    
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    return &RedisBase{
        client: client,
        logger: logger,
        config: config,
    }, nil
}

func (r *RedisBase) Close() error {
    return r.client.Close()
}

func (r *RedisBase) HealthCheck(ctx context.Context) error {
    return r.client.Ping(ctx).Err()
}

func (r *RedisBase) CollectMetrics(ctx context.Context) map[string]interface{} {
    stats := r.client.PoolStats()
    return map[string]interface{}{
        "total_conns": stats.TotalConns,
        "idle_conns":  stats.IdleConns,
        "stale_conns": stats.StaleConns,
        "hits":        stats.Hits,
        "misses":      stats.Misses,
        "timeouts":    stats.Timeouts,
    }
}