package localization_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	"github.com/clin211/gin-enterprise-template/internal/pkg/logging"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
)

func TestInvalidPasswordMessage(t *testing.T) {
	translator := localization.New()
	for _, tt := range []struct {
		lang language.Tag
		want string
	}{
		{language.English, "Invalid password."},
		{language.SimplifiedChinese, "密码无效。"},
	} {
		ctx := i18n.WithContext(context.Background(), translator.Select(tt.lang))
		got := errno.ErrPasswordInvalid.Localize(ctx)
		if got.Message != tt.want || got.Code != errorsx.CodeUserInvalidPassword || got.Reason != "User.InvalidPassword" {
			t.Errorf("%s invalid password = %+v", tt.lang, got)
		}
	}
}

func TestTemplateFailureLogKeepsRequestContext(t *testing.T) {
	var buffer bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	slog.SetDefault(slog.New(logging.NewContextHandler(slog.NewJSONHandler(&buffer, nil))))
	ctx := contextx.WithRequestID(context.Background(), "translation-request")
	ctx = contextx.WithTraceID(ctx, "translation-trace")
	ctx = i18n.WithContext(ctx, localization.New())
	got := errorsx.ErrInvalidArgument.WithMessageID("Validation.NicknameTooLong").Localize(ctx)
	if got.Message != "Invalid request parameters." {
		t.Fatalf("fallback = %q", got.Message)
	}
	var record map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
		t.Fatalf("decode diagnostic log: %v; log = %s", err, &buffer)
	}
	if record["request_id"] != "translation-request" || record["trace_id"] != "translation-trace" {
		t.Errorf("diagnostic lost request context: %v", record)
	}
	if record["message_id"] != "Validation.NicknameTooLong" || record["language"] != "en" {
		t.Errorf("diagnostic lost translation context: %v", record)
	}
}
