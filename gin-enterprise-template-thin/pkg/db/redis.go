package db

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions 定义 Redis 数据库的配置选项.
type RedisOptions struct {
	Addr         string
	Username     string
	Password     string
	Database     int
	MaxRetries   int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	PoolSize     int
}

// NewRedis 使用给定的选项创建一个新的 Redis 数据库实例.
func NewRedis(opts *RedisOptions) (*redis.Client, error) {
	options := &redis.Options{
		Addr:         opts.Addr,
		Username:     opts.Username,
		Password:     opts.Password,
		DB:           opts.Database,
		MaxRetries:   opts.MaxRetries,
		MinIdleConns: opts.MinIdleConns,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolTimeout:  opts.PoolTimeout,
		PoolSize:     opts.PoolSize,
	}

	rdb := redis.NewClient(options)

	// 检查 Redis 是否正常工作
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return rdb, nil
}
