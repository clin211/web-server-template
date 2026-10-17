package app

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// 组件自身的兼容回退文本；业务词表由应用显式注入，不在此处加载。
//
//go:embed defaults.json
var defaultsJSON []byte

var defaults = sync.OnceValue(func() map[string]string {
	var messages map[string]string
	if err := json.Unmarshal(defaultsJSON, &messages); err != nil {
		// 内嵌资源无效属于构建产物错误，不能带着损坏的默认文案启动。
		panic(fmt.Errorf("decode embedded app defaults: %w", err))
	}
	return messages
})

func defaultMessage(id string) string {
	if message, ok := defaults()[id]; ok {
		return message
	}
	return id
}
