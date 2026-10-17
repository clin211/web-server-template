package i18n

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestParameterizedTranslations(t *testing.T) {
	translator := New(WithFS(testFS))
	for _, tt := range []struct {
		lang language.Tag
		want string
	}{
		{language.English, "Maximum 45 for nickname"},
		{language.French, "nickname : maximum 45"},
	} {
		params := Params{"MaxLength": 45, "FieldName": "nickname"}
		if got := translator.Select(tt.lang).T("example.limit", params); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.lang, got, tt.want)
		}
	}
	literal := "{{.MaxLength}}"
	if got := translator.T("example.limit", Params{"MaxLength": 45, "FieldName": literal}); got != "Maximum 45 for {{.MaxLength}}" {
		t.Errorf("parameter value was reinterpreted: %q", got)
	}
}

func TestTemplateDiagnosticsOmitParameterValues(t *testing.T) {
	var buffer bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))
	message := &goi18n.Message{ID: "example.lookup", Other: `{{index . .Lookup}}`}
	if got := New().LocalizeT(message, Params{"Lookup": "private-lookup-value"}); got != message.ID {
		t.Fatalf("fallback = %q", got)
	}
	if strings.Contains(buffer.String(), "private-lookup-value") {
		t.Errorf("diagnostic leaked a parameter: %s", &buffer)
	}
}

func TestTemplateFailuresAndFallback(t *testing.T) {
	translator := New(WithFS(testFS)).Select(language.French)
	if _, err := translator.Localize(&goi18n.Message{ID: "example.limit"}, Params{"MaxLength": 45}); err == nil {
		t.Fatal("missing template parameter was accepted")
	}
	if got := translator.LocalizeT(&goi18n.Message{ID: "example.limit", Other: "Safe fallback"}); got != "Safe fallback" {
		t.Errorf("missing parameter fallback = %q", got)
	}
	if got := translator.T("example.broken", Params{"MaxLength": 45}); got != "Maximum 45" {
		t.Errorf("English fallback = %q", got)
	}
	if got := translator.LocalizeT(&goi18n.Message{ID: "example.malformed", Other: "Safe fallback"}); got != "Safe fallback" {
		t.Errorf("malformed template fallback = %q", got)
	}
	if _, err := translator.Localize(&goi18n.Message{ID: "example.limit"}, Params{"MaxLength": []int{45}}); !errors.Is(err, ErrInvalidParams) {
		t.Errorf("non-scalar parameter error = %v", err)
	}
}

func TestStrictNativeIndex(t *testing.T) {
	translator := New(WithFS(testFS))
	for _, tt := range []struct {
		name, source string
		params       Params
		want         string
		wantErr      bool
	}{
		{"missing key", `Limit {{index . "MaxLength"}}`, Params{}, "", true},
		{"nil params", `Limit {{index . "MaxLength"}}`, nil, "", true},
		{"present key", `Limit {{index . "MaxLength"}}`, Params{"MaxLength": 45}, "Limit 45", false},
		{"zero value", `Limit {{index . "MaxLength"}}`, Params{"MaxLength": 0}, "Limit 0", false},
		{"root alias", `{{$root := .}}Limit {{index $root "MaxLength"}}`, Params{}, "", true},
		{"dynamic key", `{{index . .Key}}`, Params{"Key": "MaxLength"}, "", true},
		{"string indexing", `{{index . "Name" 1}}`, Params{"Name": "abc"}, "98", false},
		{"unsigned string index", `{{index .Name .Offset}}`, Params{"Name": "abc", "Offset": uint(1)}, "98", false},
		{"negative index", `{{index .Name -1}}`, Params{"Name": "abc"}, "", true},
		{"out of range", `{{index .Name 3}}`, Params{"Name": "abc"}, "", true},
		{"fractional index", `{{index .Name .Offset}}`, Params{"Name": "abc", "Offset": 1.5}, "", true},
		{"nonindexable value", `{{index .MaxLength 0}}`, Params{"MaxLength": 45}, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			message := &goi18n.Message{ID: "test.index", Other: tt.source}
			got, err := translator.Localize(message, tt.params)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("Localize = (%q, %v), want (%q, error=%t)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
