package localization

import (
	"encoding/json"
	"testing"
)

func TestMatchLanguage(t *testing.T) {
	for _, tt := range []struct{ header, want string }{
		{"", "en"}, {"en-US", "en"}, {"en-GB", "en"},
		{"zh", "zh-Hans"}, {"zh-CN", "zh-Hans"}, {"zh-Hans", "zh-Hans"},
		{"en;q=0.3,zh-CN;q=0.9", "zh-Hans"}, {"zh;q=0,en;q=1", "en"},
		{"fr,zh;q=0.5", "zh-Hans"}, {"fr", "en"}, {"*", "en"},
		{"zh;q=broken", "en"}, {"zh;q=0", "en"},
	} {
		t.Run(tt.header, func(t *testing.T) {
			if got := MatchLanguage(tt.header).String(); got != tt.want {
				t.Fatalf("language = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCatalogsHaveSameKeys(t *testing.T) {
	catalogs := make([]map[string]string, 0, 2)
	for _, name := range []string{"locales/en.json", "locales/zh-Hans.json"} {
		data, err := catalogFS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		catalogs = append(catalogs, catalog)
	}
	for i, catalog := range catalogs {
		for key, message := range catalog {
			if message == "" || catalogs[1-i][key] == "" {
				t.Errorf("empty or missing translation: %s", key)
			}
		}
	}
}
