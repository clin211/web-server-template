package i18n

import (
	"context"
	"embed"
	"testing"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed testdata/*.json
var testFS embed.FS

func TestInjectedTranslations(t *testing.T) {
	translator := New(WithFS(testFS))
	for _, tt := range []struct {
		lang language.Tag
		want string
	}{
		{language.English, "Hello"}, {language.French, "Bonjour"},
	} {
		if got := translator.Select(tt.lang).T("example.greeting"); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.lang, got, tt.want)
		}
	}
	// Only English contains this key: the selected language falls back to the bundle default.
	if got := translator.Select(language.French).T("example.fallback"); got != "English fallback" {
		t.Errorf("fallback = %q", got)
	}
}

func TestMissingTranslationsUseCallerFallback(t *testing.T) {
	translator := New().Select(language.French)
	if got := translator.LocalizeT(&goi18n.Message{ID: "custom.message", Other: "Custom fallback"}); got != "Custom fallback" {
		t.Fatalf("caller fallback = %q", got)
	}
	if got := translator.T("missing.message"); got != "missing.message" {
		t.Fatalf("missing message = %q", got)
	}
}

func TestNewHasNoApplicationMessages(t *testing.T) {
	translator := New()
	for _, id := range []string{"User.NotFound", "Task.Conflict", "Validation.EmailInvalid", "CLI.HelpShort"} {
		if got := translator.T(id); got != id {
			t.Errorf("New() contains application message %q: %q", id, got)
		}
	}
}

func TestContextKeepsInjectedTranslator(t *testing.T) {
	translator := New(WithFS(testFS)).Select(language.French)
	ctx := WithContext(context.Background(), translator)
	if got := FromContext(ctx).T("example.greeting"); got != "Bonjour" {
		t.Fatalf("context translation = %q", got)
	}
	if got := FromContext(context.Background()).T("User.NotFound"); got != "User.NotFound" {
		t.Fatalf("implicit application catalog = %q", got)
	}
}
