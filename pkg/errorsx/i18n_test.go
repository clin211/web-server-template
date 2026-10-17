package errorsx

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBizErrorLocalization(t *testing.T) {
	base := ErrInvalidArgument.WithMessageID("Validation.EmailInvalid")
	ctx := i18n.WithContext(context.Background(), i18n.New(i18n.WithFile("testdata/zh-Hans.json")).Select(language.SimplifiedChinese))
	for _, tt := range []struct {
		name string
		err  *BizError
		want string
	}{
		{"specific validation", base, "邮箱格式无效。"},
		{"explicit override", base.WithMessage("custom"), "custom"},
		{"translation after override", base.WithMessage("custom").WithMessageID("Validation.EmailInvalid"), "邮箱格式无效。"},
		{"unknown key", NewBizError(CodeUserNotFound, "custom.reason", "custom fallback"), "custom fallback"},
		{"reason", ErrInternal, "内部服务器错误。"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			original := *tt.err
			got := tt.err.Localize(ctx)
			if got.Message != tt.want || got.Code != original.Code || got.Reason != original.Reason {
				t.Fatalf("localized error = %+v", got)
			}
			if tt.err.Message != original.Message || tt.err.MessageID != original.MessageID {
				t.Fatal("shared error was modified")
			}
		})
	}
	if base.MessageID != "Validation.EmailInvalid" || ErrInvalidArgument.MessageID != "" {
		t.Fatal("WithMessageID modified shared error")
	}
}

func TestBizErrorDiagnosticsOmitParameterValues(t *testing.T) {
	var buffer bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))
	ctx := i18n.WithContext(context.Background(), i18n.New(i18n.WithFile("testdata/en.json")))
	err := ErrInvalidArgument.WithMessageID("example.lookup").WithMessageParams(i18n.Params{"Lookup": "private-lookup-value"})
	if got := err.Localize(ctx).Message; got != "Invalid request parameters." {
		t.Fatalf("fallback = %q", got)
	}
	if strings.Contains(buffer.String(), "private-lookup-value") {
		t.Errorf("diagnostic leaked a parameter: %s", &buffer)
	}
}

func TestBizErrorFallbackRemainsLiteral(t *testing.T) {
	ctx := i18n.WithContext(context.Background(), i18n.New(i18n.WithFile("testdata/en.json")))
	literal := `object {{printf "%s" "rewritten"}} missing`
	for _, err := range []*BizError{
		NewBizError(CodeUserNotFound, "External.NotFound", literal),
		NewBizError(CodeUserNotFound, "", literal),
		FromError(status.Error(codes.NotFound, literal)),
		ErrInvalidArgument.WithMessageID("missing.message").WithMessage(literal),
	} {
		if got := err.Localize(ctx).Message; got != literal {
			t.Errorf("fallback = %q, want literal %q", got, literal)
		}
	}
	if got := ErrInvalidArgument.WithMessageID("missing.message").Localize(ctx).Message; got != "Invalid request parameters." {
		t.Errorf("missing message reason fallback = %q", got)
	}
}

func TestBizErrorParamsAreSnapshots(t *testing.T) {
	translator := i18n.New(i18n.WithFile("testdata/en.json"))
	ctx := i18n.WithContext(context.Background(), translator)
	params := i18n.Params{"MaxLength": 45}
	base := ErrInvalidArgument.WithMessageID("example.limit")
	first := base.WithMessageParams(params)
	params["MaxLength"] = 99
	second := first.WithMessageParams(i18n.Params{"MaxLength": 60})
	for _, tt := range []struct {
		name string
		err  *BizError
		want string
	}{
		{"snapshot", first, "Limit 45"},
		{"replacement", second, "Limit 60"},
		{"other derivation", first.WithDetails("details"), "Limit 45"},
		{"explicit message", first.WithMessage("custom"), "custom"},
		{"new message clears params", first.WithMessageID("example.limit"), "Invalid request parameters."},
		{"nil clears params", first.WithMessageParams(nil), "Invalid request parameters."},
		{"invalid parameter", base.WithMessageParams(i18n.Params{"MaxLength": []int{45}}), "Invalid request parameters."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Localize(ctx).Message; got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
		})
	}
	if got := base.Localize(ctx).Message; got != "Invalid request parameters." {
		t.Errorf("base changed: %q", got)
	}
}

func TestBizErrorBrokenTranslationFallsBackToDefaultLanguage(t *testing.T) {
	translator := i18n.New(i18n.WithFile("testdata")).Select(language.SimplifiedChinese)
	ctx := i18n.WithContext(context.Background(), translator)
	err := ErrInvalidArgument.WithMessageID("example.limit").WithMessageParams(i18n.Params{"MaxLength": 45})
	if got := err.Localize(ctx).Message; got != "Limit 45" {
		t.Errorf("default language fallback = %q", got)
	}
}
