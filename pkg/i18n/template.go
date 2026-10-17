package i18n

import (
	"errors"
	"fmt"
	"reflect"
	"text/template"

	i18ntemplate "github.com/nicksnyder/go-i18n/v2/i18n/template"
)

type strictTextParser struct{}

var errInvalidIndex = errors.New("invalid template index")

func (strictTextParser) Parse(source, leftDelim, rightDelim string) (i18ntemplate.ParsedTemplate, error) {
	parser := i18ntemplate.TextParser{
		Option: "missingkey=error",
		Funcs:  template.FuncMap{"index": strictIndex},
	}
	return parser.Parse(source, leftDelim, rightDelim)
}

func (strictTextParser) Cacheable() bool {
	// 函数表固定且不捕获请求数据，解析后的模板可以并发执行和缓存。
	return true
}

// strictIndex 保留 Params 能产生的原生索引行为（参数表、字符串），
// 并补上 missingkey=error 对内置 index 不生效的检查。
func strictIndex(item any, indexes ...any) (any, error) {
	for _, index := range indexes {
		value, err := strictIndexValue(item, index)
		if err != nil {
			return nil, err
		}
		item = value
	}
	return item, nil
}

func strictIndexValue(item, index any) (any, error) {
	switch value := item.(type) {
	case map[string]any:
		key, ok := index.(string)
		if !ok {
			return nil, fmt.Errorf("%w: map index must be a string, got %T", errInvalidIndex, index)
		}
		result, ok := value[key]
		if !ok {
			return nil, fmt.Errorf("%w: missing map key %q", errInvalidIndex, key)
		}
		return result, nil
	case string:
		offset, err := stringIndex(index, len(value))
		if err != nil {
			return nil, err
		}
		return value[offset], nil
	default:
		return nil, fmt.Errorf("%w: cannot index %T", errInvalidIndex, item)
	}
}

func stringIndex(index any, length int) (int, error) {
	value := reflect.ValueOf(index)
	var offset uint64
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() < 0 {
			return 0, fmt.Errorf("%w: string index out of range", errInvalidIndex)
		}
		offset = uint64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		offset = value.Uint()
	default:
		return 0, fmt.Errorf("%w: string index must be an integer, got %T", errInvalidIndex, index)
	}
	if offset >= uint64(length) {
		return 0, fmt.Errorf("%w: string index out of range", errInvalidIndex)
	}
	return int(offset), nil
}
