package token

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const refreshTokenKeyPrefix = "auth:refresh:used:"
const loginAttemptKeyPrefix = "auth:login:attempts:"

var (
	errInvalidLoginRateLimitInput = errors.New("invalid login rate limit input")
	incrementLoginAttempts        = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 or redis.call("PTTL", KEYS[1]) < 0 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`)
)

// RedisRefreshTokenStore 使用 Redis SETNX 实现跨实例 refresh token 单次消费。
type RedisRefreshTokenStore struct {
	client *redis.Client
}

// NewRedisRefreshTokenStore 创建 Redis refresh token 存储。
func NewRedisRefreshTokenStore(client *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{client: client}
}

// Consume 首次消费 token ID 时返回 true；已被其他请求消费时返回 false。
func (s *RedisRefreshTokenStore) Consume(ctx context.Context, tokenID string, expiration time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, ErrRefreshTokenStoreRequired
	}
	consumed, err := s.client.SetNX(ctx, refreshTokenKeyPrefix+tokenID, "1", expiration).Result()
	if err != nil {
		return false, fmt.Errorf("store consumed refresh token: %w", err)
	}
	return consumed, nil
}

// Ping 检查 Redis 是否仍可用于 refresh token 单次消费。
func (s *RedisRefreshTokenStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return ErrRefreshTokenStoreRequired
	}
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping refresh token Redis: %w", err)
	}
	return nil
}

// AllowLogin 使用 Redis 原子计数实现跨实例固定窗口限流。identity 在写入 Redis
// key 前经过 SHA-256，避免把客户端地址等原始标识暴露在运维 key 列表中。
func (s *RedisRefreshTokenStore) AllowLogin(ctx context.Context, identity string, maximum int64, window time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, ErrRefreshTokenStoreRequired
	}
	if identity == "" || maximum <= 0 || window <= 0 {
		return false, errInvalidLoginRateLimitInput
	}
	digest := sha256.Sum256([]byte(identity))
	key := fmt.Sprintf("%s%x", loginAttemptKeyPrefix, digest[:])
	count, err := incrementLoginAttempts.Run(ctx, s.client, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("increment login attempts: %w", err)
	}
	return count <= maximum, nil
}

// Close 关闭底层 Redis 客户端。
func (s *RedisRefreshTokenStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
