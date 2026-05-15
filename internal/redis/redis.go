package redis

import (
	"context"
	"event-platform/internal/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// 全局Redis客户端实例
var client *redis.Client

// NewRedisClient 创建Redis客户端连接
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %v", err)
	}

	logrus.Info("Redis连接成功")
	return client, nil
}

// GetClient 获取Redis客户端实例
func GetClient() *redis.Client {
	return client
}
