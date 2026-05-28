package job

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

var (
	// ErrLockNotAcquired 表示指定锁已被其他执行者持有。
	ErrLockNotAcquired = errors.New("failed to acquire lock")
	// ErrLockNotHeld 表示释放或续期时锁不存在或 token 不匹配。
	ErrLockNotHeld = errors.New("lock is not held")
)

// Locker 定义任务调度使用的分布式锁接口。
type Locker interface {
	// Acquire 获取指定 key 的锁，并返回用于释放和续期的 token。
	Acquire(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Release 使用 token 释放指定 key 的锁。
	Release(ctx context.Context, key string, token string) error
	// Extend 使用 token 延长指定 key 的锁有效期。
	Extend(ctx context.Context, key string, token string, ttl time.Duration) error
}

// RedisLock 基于 Redis SETNX 和 Lua 脚本实现分布式锁。
type RedisLock struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisLockWithClient 使用现有 Redis 客户端创建 RedisLock。
func NewRedisLockWithClient(client *redis.Client, opts *genericoptions.JobOptions) *RedisLock {
	var ttl = 2 * time.Minute
	if opts != nil && opts.Scheduler.LockTTL > 0 {
		ttl = opts.Scheduler.LockTTL
	}
	return &RedisLock{client: client, ttl: ttl}
}

var unlockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`)

// Acquire 获取指定 key 的锁，并返回用于释放和续期的 token。
func (l *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if l == nil || l.client == nil {
		return "", errors.New("redis lock is not initialized")
	}

	if ttl <= 0 {
		ttl = l.ttl
	}

	token := uuid.New().String()
	result, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", err
	}
	if !result {
		return "", ErrLockNotAcquired
	}
	return token, nil
}

// Release 使用 token 释放指定 key 的锁。
func (l *RedisLock) Release(ctx context.Context, key string, token string) error {
	if l == nil || l.client == nil {
		return nil
	}
	if key == "" || token == "" {
		return ErrLockNotHeld
	}

	result, err := unlockScript.Run(ctx, l.client, []string{key}, token).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// Extend 使用 token 延长指定 key 的锁有效期。
func (l *RedisLock) Extend(ctx context.Context, key string, token string, ttl time.Duration) error {
	if l == nil || l.client == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = l.ttl
	}

	script := redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
else
    return 0
end
`)

	result, err := script.Run(ctx, l.client, []string{key}, token, int64(ttl.Milliseconds())).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}
