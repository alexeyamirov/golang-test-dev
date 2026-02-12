package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

// CacheMetrics кэширует метрики
func (r *RedisCache) CacheMetrics(ctx context.Context, key string, metrics []MetricValue, ttl time.Duration) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetCachedMetrics получает метрики из кэша
func (r *RedisCache) GetCachedMetrics(ctx context.Context, key string) ([]MetricValue, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var metrics []MetricValue
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}
	return metrics, nil
}

// CacheAlertStats кэширует статистику алертов
func (r *RedisCache) CacheAlertStats(ctx context.Context, key string, stats *AlertStats, ttl time.Duration) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetCachedAlertStats получает статистику алертов из кэша
func (r *RedisCache) GetCachedAlertStats(ctx context.Context, key string) (*AlertStats, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var stats AlertStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
