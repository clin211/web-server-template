package task

import "testing"

func TestReportCSVFormulaEscaping(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"plain", "plain"}, {"=1+1", "'=1+1"}, {" \t=SUM(A1)", "' \t=SUM(A1)"}, {"\r\n@SUM(A1)", "'\r\n@SUM(A1)"}, {"+1", "'+1"}, {"-1", "'-1"}, {"", ""}} {
		if got := safeCSV(tc.input); got != tc.want {
			t.Errorf("safeCSV(%q)=%q want %q", tc.input, got, tc.want)
		}
	}
}
