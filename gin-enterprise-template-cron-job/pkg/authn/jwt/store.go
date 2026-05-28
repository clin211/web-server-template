package jwt

import (
	"context"
	"time"
)

// Storer 令牌存储接口。
type Storer interface {
	// 存储令牌数据并指定过期时间。
	Set(ctx context.Context, accessToken string, expiration time.Duration) error

	// 从存储中删除令牌数据。
	Delete(ctx context.Context, accessToken string) (bool, error)

	// 检查令牌是否存在。
	Check(ctx context.Context, accessToken string) (bool, error)

	// 关闭存储连接。
	Close() error
}
