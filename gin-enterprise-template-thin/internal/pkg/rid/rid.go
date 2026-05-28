package rid

import (
	"github.com/clin211/gin-enterprise-template/pkg/id"
)

const defaultABC = "abcdefghijklmnopqrstuvwxyz1234567890"

type ResourceID string

const (
	// UserID 定义用户的资源标识符。
	UserID ResourceID = "user"
)

// String 将资源标识符转换为字符串。
func (rid ResourceID) String() string {
	return string(rid)
}

// New 创建带有前缀的唯一标识符。
func (rid ResourceID) New(counter uint64) string {
	// 使用自定义选项生成唯一标识符。
	uniqueStr := id.NewCode(
		counter,
		id.WithCodeChars([]rune(defaultABC)),
		id.WithCodeL(6),
		id.WithCodeSalt(Salt()),
	)
	return rid.String() + "-" + uniqueStr
}
