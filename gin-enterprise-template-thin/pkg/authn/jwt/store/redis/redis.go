package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 包含必要的 Redis 选项。
type Config struct {
	Addr     string
	Username string
	Password string
	Database int
	// 存储键前缀。
	KeyPrefix string
}

// Store Redis 存储。
type Store struct {
	cli    *redis.Client
	prefix string
}

// NewStore 创建一个 *Store 实例来处理令牌的存储、删除和检查。
func NewStore(cfg *Config) *Store {
	// 这里不使用 `github.com/clin211/gin-enterprise-template/pkg/db` 的原因是
	// 最小化依赖，并且使用 `github.com/redis/go-redis/v9` 来
	// 创建 Redis 客户端并不复杂。
	cli := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		DB:       cfg.Database,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	return &Store{cli: cli, prefix: cfg.KeyPrefix}
}

// wrapperKey 用于构建 Redis 中的键名。
func (s *Store) wrapperKey(key string) string {
	return fmt.Sprintf("%s%s", s.prefix, key)
}

// Set 调用 Redis 客户端设置带有过期时间的键值对，
// 其中键名格式为 <prefix><accessToken>。
func (s *Store) Set(ctx context.Context, accessToken string, expiration time.Duration) error {
	cmd := s.cli.Set(ctx, s.wrapperKey(accessToken), "1", expiration)
	return cmd.Err()
}

// Delete 删除 Redis 中指定的 JWT 令牌。
func (s *Store) Delete(ctx context.Context, accessToken string) (bool, error) {
	cmd := s.cli.Del(ctx, s.wrapperKey(accessToken))
	if err := cmd.Err(); err != nil {
		return false, err
	}
	return cmd.Val() > 0, nil
}

// Check 检查 Redis 中是否存在指定的 JWT 令牌。
func (s *Store) Check(ctx context.Context, accessToken string) (bool, error) {
	cmd := s.cli.Exists(ctx, s.wrapperKey(accessToken))
	if err := cmd.Err(); err != nil {
		return false, err
	}
	return cmd.Val() > 0, nil
}

// Close 用于关闭 Redis 客户端。
func (s *Store) Close() error {
	return s.cli.Close()
}
