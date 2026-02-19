// Package database обеспечивает доступ к PostgreSQL (метрики, алерты) и Redis (кэш).
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache — кэш метрик и статистики алертов в Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache подключается к Redis и возвращает кэш.
func NewRedisCache(addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,    // адрес Redis (например localhost:6379)
		Password:     "",     // без пароля
		DB:           0,      // база по умолчанию
		PoolSize:     10,     // размер пула соединений
		MinIdleConns: 5,      // минимальное число простаивающих соединений
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// Close закрывает соединение с Redis.
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// CacheMetrics кэширует метрики в Redis с заданным TTL.
func (r *RedisCache) CacheMetrics(ctx context.Context, key string, metrics []MetricValue, ttl time.Duration) error {
	data, err := json.Marshal(metrics) // сериализуем в JSON
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err() // сохраняем с временем жизни
}

// GetCachedMetrics получает метрики из кэша. nil,nil — ключ отсутствует.
func (r *RedisCache) GetCachedMetrics(ctx context.Context, key string) ([]MetricValue, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil { // ключ не найден
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

// CacheAlertStats кэширует статистику алертов (среднее, количество) с TTL.
func (r *RedisCache) CacheAlertStats(ctx context.Context, key string, stats *AlertStats, ttl time.Duration) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetCachedAlertStats получает статистику алертов из кэша. nil,nil — ключ отсутствует.
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
