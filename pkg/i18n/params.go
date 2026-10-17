package i18n

import (
	"errors"
	"fmt"
)

// Params 是平面的模板参数。值仅支持内置字符串、布尔和数值类型，
// 不接受可变容器、指针或函数，便于业务错误复制参数并隔离并发请求。
type Params map[string]any

// ErrInvalidParams 表示参数数量或值类型不符合模板参数约定。
var ErrInvalidParams = errors.New("invalid localization parameters")

func validateParams(params Params) error {
	for key, value := range params {
		switch value.(type) {
		case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		default:
			// 只记录键和类型，不将参数值写入错误或日志。
			return fmt.Errorf("%w: key %q has unsupported type %T", ErrInvalidParams, key, value)
		}
	}
	return nil
}

func optionalParams(params []Params) (Params, error) {
	if len(params) > 1 {
		return nil, fmt.Errorf("%w: expected at most one parameter map", ErrInvalidParams)
	}
	if len(params) == 1 {
		return params[0], nil
	}
	return Params{}, nil
}
