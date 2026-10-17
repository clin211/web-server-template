package job

import "errors"

var errDaysMustBeBetween1And90 = errors.New("days must be between 1 and 90")

// ReportParams defines the version-one report request.
type ReportParams struct {
	Days int `json:"days"`
}

// ValidateReport limits query work and rejects unsupported ranges.
func ValidateReport(p ReportParams) error {
	if p.Days < 1 || p.Days > 90 {
		return Permanent(errDaysMustBeBetween1And90)
	}
	return nil
}
