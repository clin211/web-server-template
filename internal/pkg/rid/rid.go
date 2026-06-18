package rid

import (
	"context"

	"github.com/clin211/gin-enterprise-template/pkg/id"
)

const defaultCharset = "abcdefghijklmnopqrstuvwxyz1234567890"

type ResourceID string

const (
	// UserID 定义用户资源的业务 ID 前缀。
	UserID ResourceID = "user"
)

var defaultIDCounter = id.NewSonyflake()

// String 返回资源 ID 前缀的字符串表示。
func (rid ResourceID) String() string {
	return string(rid)
}

// New 使用给定计数器创建带前缀的业务 ID。
func (rid ResourceID) New(counter uint64) string {
	uniqueStr := id.NewCode(
		counter,
		id.WithCodeChars([]rune(defaultCharset)),
		id.WithCodeL(6),
		id.WithCodeSalt(Salt()),
	)
	return rid.String() + "-" + uniqueStr
}

// MustNew 使用默认计数器创建带前缀的业务 ID。
func (rid ResourceID) MustNew() string {
	return rid.New(defaultIDCounter.Id(context.Background()))
}
