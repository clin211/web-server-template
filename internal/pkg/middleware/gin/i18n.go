package gin

import (
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	ginframework "github.com/gin-gonic/gin"
)

// I18n 将请求语言及翻译器注入上下文。translator 必须在注册中间件前加载完毕，
// 服务期间仅共享只读 Bundle，各请求使用独立 Localizer，避免语言串扰。
func I18n(translator *i18n.I18n) ginframework.HandlerFunc {
	return func(c *ginframework.Context) {
		selected := translator.Select(localization.MatchLanguage(c.GetHeader("Accept-Language")))
		c.Request = c.Request.WithContext(i18n.WithContext(c.Request.Context(), selected))
		c.Header("Content-Language", selected.Language().String())
		c.Writer.Header().Add("Vary", "Accept-Language")
		c.Next()
	}
}
