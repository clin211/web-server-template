package errcheck

import "testing"

func TestAnalyzerUsesSupportedFlags(t *testing.T) {
	if Analyzer == nil || Analyzer.Run == nil {
		t.Fatal("analyzer must be usable")
	}
	flag := Analyzer.Flags.Lookup("exclude")
	if flag == nil || flag.Value.String() != "" {
		t.Fatal("use the upstream default exclusion configuration")
	}
}
