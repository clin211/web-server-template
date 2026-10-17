package localization

import (
	"testing"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

func TestBusinessMessageParameters(t *testing.T) {
	translator := New()
	for _, tt := range []struct {
		id     string
		params i18n.Params
		en, zh string
	}{
		{"Validation.NicknameTooLong", i18n.Params{"MaxLength": 45}, "nickname must be less than 45 characters", "昵称长度必须小于 45 个字符。"},
		{"Validation.PasswordTooShort", i18n.Params{"MinLength": 8}, "password must be at least 8 characters long", "密码长度至少为 8 个字符。"},
		{"Validation.UsernameInvalid", i18n.Params{"MinLength": 4, "MaxLength": 24}, "Invalid username: use 4 to 24 letters, numbers or underscores.", "用户名无效：用户名只能包含字母、数字和下划线，长度必须在 4 到 24 个字符之间。"},
		{"Validation.LimitPositive", i18n.Params{"Min": 0}, "limit must be greater than 0", "每页数量必须大于 0。"},
		{"Validation.LimitPositive", i18n.Params{"Min": 5}, "limit must be greater than 5", "每页数量必须大于 5。"},
		{"Task.IdempotencyKeyInvalid", i18n.Params{"MaxLength": 256}, "Idempotency-Key required, at most 256 characters", "必须提供 Idempotency-Key，且长度不能超过 256 个字符。"},
	} {
		for _, lang := range []language.Tag{language.English, language.SimplifiedChinese} {
			t.Run(tt.id+lang.String(), func(t *testing.T) {
				want := tt.en
				if lang == language.SimplifiedChinese {
					want = tt.zh
				}
				if got := translator.Select(lang).T(tt.id, tt.params); got != want {
					t.Errorf("message = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestDefaultMessageRendersParameters(t *testing.T) {
	if got := DefaultMessage("Validation.NicknameTooLong", i18n.Params{"MaxLength": 45}); got != "昵称长度必须小于 45 个字符。" {
		t.Fatalf("default = %q", got)
	}
	if got := DefaultMessage("Validation.NicknameTooLong"); got != "Validation.NicknameTooLong" {
		t.Fatalf("missing parameters exposed raw template: %q", got)
	}
}
