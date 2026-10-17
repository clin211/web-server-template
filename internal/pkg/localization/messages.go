package localization

import (
	"sync"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

// 初始化完成后不再修改资源或语言，所有调用仅执行只读翻译。
var defaultTranslator = sync.OnceValue(func() *i18n.I18n {
	return New().Select(language.SimplifiedChinese)
})

// DefaultMessage 渲染中文默认消息，可提供一个参数表；资源缺失时按正常语言规则回退。
// 模板参数缺失时返回消息 ID，不暴露未渲染的模板。
func DefaultMessage(id string, params ...i18n.Params) string {
	return defaultTranslator().T(id, params...)
}
