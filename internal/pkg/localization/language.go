package localization

import "golang.org/x/text/language"

// MatchLanguage 按请求头权重选择项目支持的语言；无匹配或无效时使用英文。
func MatchLanguage(header string) language.Tag {
	tags, _, err := language.ParseAcceptLanguage(header)
	if err == nil {
		for _, tag := range tags {
			base, _ := tag.Base()
			switch base.String() {
			case "zh":
				return language.SimplifiedChinese
			case "en":
				return language.English
			}
		}
	}
	return language.English
}
