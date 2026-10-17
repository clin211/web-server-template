package i18n

import (
	"context"
)

type translator struct{}

// WithContext 将请求翻译器注入上下文。
func WithContext(ctx context.Context, i *I18n) context.Context {
	return context.WithValue(ctx, translator{}, i)
}

// FromContext 获取请求翻译器；未注入时返回不含资源的翻译器，使用调用方提供的回退文案。
func FromContext(ctx context.Context) *I18n {
	if i, ok := ctx.Value(translator{}).(*I18n); ok && i != nil {
		return i
	}

	return New()
}
