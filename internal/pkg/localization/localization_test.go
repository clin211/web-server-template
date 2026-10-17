package localization

import (
	"os"
	"path/filepath"
	"testing"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestEmbeddedTranslations(t *testing.T) {
	translator := New()
	for _, tt := range []struct {
		lang language.Tag
		want string
	}{
		{language.English, "User not found."},
		{language.AmericanEnglish, "User not found."},
		{language.SimplifiedChinese, "用户未找到。"},
		{language.Chinese, "用户未找到。"},
	} {
		t.Run(tt.lang.String(), func(t *testing.T) {
			if got := translator.Select(tt.lang).T("User.NotFound"); got != tt.want {
				t.Fatalf("translation = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLocalizationFallback(t *testing.T) {
	translator := New().Select(language.French)
	if got := translator.T("User.NotFound"); got != "User not found." {
		t.Fatalf("English fallback = %q", got)
	}
	if got := translator.LocalizeT(&goi18n.Message{ID: "custom.message", Other: "Custom fallback"}); got != "Custom fallback" {
		t.Fatalf("caller fallback = %q", got)
	}
	if got := translator.T("missing.message"); got != "missing.message" {
		t.Fatalf("missing message = %q", got)
	}
}

func TestMissingChineseTranslationFallsBackToEnglish(t *testing.T) {
	file := filepath.Join(t.TempDir(), "en.json")
	if err := os.WriteFile(file, []byte(`{"custom.message":"English catalog"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	translator := New()
	translator.Add(file)
	translator = translator.Select(language.SimplifiedChinese)
	message := &goi18n.Message{ID: "custom.message", Other: "Caller fallback"}
	if got := translator.LocalizeT(message); got != "English catalog" {
		t.Fatalf("missing Chinese translation = %q, want English catalog", got)
	}
}
