// Package localization 提供当前项目的语言资源与语言选择策略。
// 通用翻译机制由 pkg/i18n 提供，公共包不依赖本包。
package localization

import (
	"embed"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var catalogFS embed.FS

// New 构造默认英文的项目翻译器。所有资源在返回前加载完成，
// 运行期间仅共享只读 Bundle，请求使用 Select 创建独立 Localizer。
func New() *i18n.I18n {
	return i18n.New(i18n.WithLanguage(language.English), i18n.WithFS(catalogFS))
}
