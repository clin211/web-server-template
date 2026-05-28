package authn

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// IToken 定义了实现通用令牌的方法。
type IToken interface {
	// 获取令牌字符串。
	GetToken() string
	// 获取令牌类型。
	GetTokenType() string
	// 获取令牌过期时间戳。
	GetExpiresAt() int64
	// JSON 编码。
	EncodeToJSON() ([]byte, error)
}

// Authenticator 定义了用于令牌处理的方法。
type Authenticator interface {
	// Sign 用于生成令牌。
	Sign(ctx context.Context, userID string) (IToken, error)

	// Destroy 用于销毁令牌。
	Destroy(ctx context.Context, accessToken string) error

	// ParseClaims 解析令牌并返回声明。
	ParseClaims(ctx context.Context, accessToken string) (*jwt.RegisteredClaims, error)

	// Release 用于释放请求的资源。
	Release() error
}

// Encrypt 使用 bcrypt 对明文进行加密。
func Encrypt(source string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(source), bcrypt.DefaultCost)
	return string(hashedBytes), err
}

// Compare 比较加密后的文本与明文是否相同。
func Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
