package validation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

func TestCreateUserValidationTranslations(t *testing.T) {
	validator := &Validator{}
	translator := localization.New()
	for _, tt := range []struct {
		name   string
		change func(*v1.CreateUserRequest)
		en, zh string
	}{
		{"password missing", func(r *v1.CreateUserRequest) { r.Password = "" }, "password cannot be empty", "密码不能为空。"},
		{"password short", func(r *v1.CreateUserRequest) { r.Password = "a1" }, "password must be at least 6 characters long", "密码长度至少为 6 个字符。"},
		{"Unicode password short", func(r *v1.CreateUserRequest) { r.Password = "a1😀" }, "password must be at least 6 characters long", "密码长度至少为 6 个字符。"},
		{"password letter", func(r *v1.CreateUserRequest) { r.Password = "123456" }, "password must contain at least one letter", "密码必须包含至少一个字母。"},
		{"password number", func(r *v1.CreateUserRequest) { r.Password = "abcdef" }, "password must contain at least one number", "密码必须包含至少一个数字。"},
		{"email missing", func(r *v1.CreateUserRequest) { r.Email = "" }, "email cannot be empty", "邮箱不能为空。"},
		{"email invalid", func(r *v1.CreateUserRequest) { r.Email = "invalid" }, "invalid email format", "邮箱格式无效。"},
		{"phone missing", func(r *v1.CreateUserRequest) { r.Phone = "" }, "phone cannot be empty", "手机号不能为空。"},
		{"phone invalid", func(r *v1.CreateUserRequest) { r.Phone = "123" }, "invalid phone format", "手机号格式无效。"},
		{"nickname too long", func(r *v1.CreateUserRequest) { s := strings.Repeat("a", 30); r.Nickname = &s }, "nickname must be less than 30 characters", "昵称长度必须小于 30 个字符。"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := &v1.CreateUserRequest{Username: "user1", Password: "pass123", Email: "test@example.com", Phone: "13800138000"}
			tt.change(request)
			err := validator.ValidateCreateUserRequest(context.Background(), request)
			var bizErr *errorsx.BizError
			if !errors.As(err, &bizErr) {
				t.Fatalf("error = %v", err)
			}
			if bizErr.Reason != "Common.InvalidArgument" || bizErr.Code != errorsx.CodeUserInvalidCredentials {
				t.Fatalf("contract changed: %v", bizErr)
			}
			for _, lang := range []language.Tag{language.English, language.SimplifiedChinese} {
				ctx := i18n.WithContext(context.Background(), translator.Select(lang))
				want := tt.en
				if lang == language.SimplifiedChinese {
					want = tt.zh
				}
				if got := bizErr.Localize(ctx).Message; got != want {
					t.Errorf("%s message = %q, want %q", lang, got, want)
				}
			}
		})
	}
}

func TestUserValidationBoundaryMessages(t *testing.T) {
	validator := &Validator{}
	ctx := i18n.WithContext(context.Background(), localization.New())
	for _, tt := range []struct {
		name   string
		change func(*v1.CreateUserRequest)
		want   string
	}{
		{"nickname below maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("a", 29); r.Nickname = &s }, ""},
		{"nickname at maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("a", 30); r.Nickname = &s }, "nickname must be less than 30 characters"},
		{"Chinese nickname ten characters", func(r *v1.CreateUserRequest) { s := strings.Repeat("中", 10); r.Nickname = &s }, ""},
		{"Chinese nickname below maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("中", 29); r.Nickname = &s }, ""},
		{"Chinese nickname at maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("中", 30); r.Nickname = &s }, "nickname must be less than 30 characters"},
		{"emoji nickname below maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("😀", 29); r.Nickname = &s }, ""},
		{"emoji nickname at maximum", func(r *v1.CreateUserRequest) { s := strings.Repeat("😀", 30); r.Nickname = &s }, "nickname must be less than 30 characters"},
		{"username too short", func(r *v1.CreateUserRequest) { r.Username = "ab" }, "Invalid username: use 3 to 20 letters, numbers or underscores."},
		{"username at minimum", func(r *v1.CreateUserRequest) { r.Username = "abc" }, ""},
		{"username at maximum", func(r *v1.CreateUserRequest) { r.Username = strings.Repeat("a", 20) }, ""},
		{"username too long", func(r *v1.CreateUserRequest) { r.Username = strings.Repeat("a", 21) }, "Invalid username: use 3 to 20 letters, numbers or underscores."},
		{"password too short", func(r *v1.CreateUserRequest) { r.Password = "abc12" }, "password must be at least 6 characters long"},
		{"password at minimum", func(r *v1.CreateUserRequest) { r.Password = "abc123" }, ""},
		{"Unicode password below minimum", func(r *v1.CreateUserRequest) { r.Password = "a1中😀文" }, "password must be at least 6 characters long"},
		{"Unicode password at minimum", func(r *v1.CreateUserRequest) { r.Password = "a1中😀文好" }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := &v1.CreateUserRequest{Username: "user1", Password: "pass123", Email: "test@example.com", Phone: "13800138000"}
			tt.change(request)
			err := validator.ValidateCreateUserRequest(ctx, request)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("valid boundary rejected: %v", err)
				}
				return
			}
			var bizErr *errorsx.BizError
			if !errors.As(err, &bizErr) {
				t.Fatalf("error = %v", err)
			}
			if got := bizErr.Localize(ctx).Message; got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
}
