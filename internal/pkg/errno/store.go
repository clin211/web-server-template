package errno

import (
	"errors"

	"gorm.io/gorm"
)

// FromStore 将数据访问层错误翻译为稳定的业务错误。
// 目前仅翻译"记录不存在"（gorm.ErrRecordNotFound，含包装链）；其余错误原样
// 返回，由 errorsx.FromError 统一兜底为系统内部错误，避免向客户端泄露底层
// 实现细节。Biz 层在 Get/Delete 等以存在性为前置的路径上应统一经过本函数，
// 防止"资源不存在"被误报为 500。
func FromStore(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
