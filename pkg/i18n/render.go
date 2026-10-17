package i18n

import (
	"errors"
	"fmt"
	"log/slog"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// Localize 使用 go-i18n 选择消息并严格渲染命名参数，模板使用 {{.ParamName}}。
// 缺失语言的正常回退视为成功；缺失参数、无效模板或不支持的参数类型返回错误。
func (i I18n) Localize(message *goi18n.Message, params Params) (string, error) {
	if message == nil {
		return "", nil
	}
	if err := validateParams(params); err != nil {
		return "", err
	}
	result, err := i.localizer.Localize(&goi18n.LocalizeConfig{
		DefaultMessage: message,
		TemplateData:   map[string]any(params),
		TemplateParser: strictTextParser{},
	})
	var missing *goi18n.MessageNotFoundErr
	if err == nil || (errors.As(err, &missing) && result != "") {
		return result, nil
	}
	return "", fmt.Errorf("localize %q: %w", message.ID, err)
}

// LocalizeT 渲染消息，可提供一个参数表；旧的无参数调用保持兼容。
// 渲染失败时记录诊断并尝试默认语言、调用方回退文本，最后返回消息 ID，
// 不返回未完成的模板或 <no value>。参数值只作为数据，不进行二次模板解析。
func (i I18n) LocalizeT(message *goi18n.Message, params ...Params) string {
	if message == nil {
		return ""
	}
	data, err := optionalParams(params)
	if err == nil {
		var result string
		result, err = i.Localize(message, data)
		if err == nil {
			return result
		}
	}
	return i.localizeFallback(message, data, err)
}

func (i I18n) localizeFallback(message *goi18n.Message, data Params, err error) string {
	// 原生模板错误可能包含参数值（例如动态索引键），日志只记录资源定位信息。
	slog.Warn("localize message failed", "message_id", message.ID, "language", i.lang.String())
	if errors.Is(err, ErrInvalidParams) {
		data = nil
	}
	if i.lang != i.ops.language {
		if result, fallbackErr := i.Select(i.ops.language).Localize(message, data); fallbackErr == nil {
			return result
		}
	}
	// 空资源翻译器只渲染调用方 DefaultMessage，避免再次选择出错的资源模板。
	if message.Other != "" {
		if result, fallbackErr := New(WithLanguage(i.ops.language)).Localize(message, data); fallbackErr == nil {
			return result
		}
	}
	return message.ID
}

// LocalizeE 是 LocalizeT 的错误包装器，可提供一个参数表。
func (i I18n) LocalizeE(message *goi18n.Message, params ...Params) error {
	return &localizedError{message: i.LocalizeT(message, params...)}
}

// T 按消息 ID 翻译，可提供一个参数表。
func (i I18n) T(id string, params ...Params) string {
	return i.LocalizeT(&goi18n.Message{ID: id}, params...)
}

// E 是 T 的错误包装器，可提供一个参数表。
func (i I18n) E(id string, params ...Params) error {
	return &localizedError{message: i.T(id, params...)}
}
