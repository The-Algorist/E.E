package repository

import "time"

type RedisConfig struct {
    URL            string
    Password       string
    DB             int
    MaxRetries     int
    RetryBackoff   time.Duration
    MinIdleConns   int
    PoolSize       int
    PoolTimeout    time.Duration
	ConnectTimeout time.Duration
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    JobTTL         time.Duration
}

func DefaultRedisConfig() RedisConfig {
    return RedisConfig{
        URL:            "localhost:6379",
        DB:             0,
        MaxRetries:     3,
        RetryBackoff:   time.Millisecond * 100,
        MinIdleConns:   10,
        PoolSize:       100,
        PoolTimeout:    time.Second * 30,
		ConnectTimeout: time.Second * 5,
        // DialTimeout:    time.Second * 5,
        ReadTimeout:    time.Second * 3,
        WriteTimeout:   time.Second * 3,
        JobTTL:         time.Hour * 24,
    }
}