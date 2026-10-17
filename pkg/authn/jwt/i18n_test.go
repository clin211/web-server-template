package jwt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

func TestTokenErrorsUseContextLanguage(t *testing.T) {
	translator := i18n.New(i18n.WithFile("testdata/zh-Hans.json"))
	auth := New(nil, WithExpired(-time.Hour))
	info, err := auth.Sign(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ token, en, zh string }{
		{"", "Token is invalid", "令牌无效。"},
		{"invalid-token", "Token is invalid", "令牌无效。"},
		{info.GetToken(), "Token has expired", "令牌已过期。"},
	} {
		for _, lang := range []language.Tag{language.English, language.SimplifiedChinese} {
			t.Run(tt.en+lang.String()+tt.token, func(t *testing.T) {
				ctx := i18n.WithContext(context.Background(), translator.Select(lang))
				_, err := auth.ParseClaims(ctx, tt.token)
				var unauthorized *UnauthorizedError
				if !errors.As(err, &unauthorized) {
					t.Fatalf("error = %v", err)
				}
				want := tt.en
				if lang == language.SimplifiedChinese {
					want = tt.zh
				}
				if unauthorized.Msg != want {
					t.Fatalf("message = %q, want %q", unauthorized.Msg, want)
				}
			})
		}
	}
}
