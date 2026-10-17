package job

import (
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
)

var errCronMustContainFiveFields = errors.New("cron must contain five fields")

// ParseSchedule accepts five-field Cron and a separate IANA location.
func ParseSchedule(spec, zone string) (cron.Schedule, error) {
	if len(strings.Fields(spec)) != 5 {
		return nil, errCronMustContainFiveFields
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	s, err := p.Parse(spec)
	if err != nil {
		return nil, fmt.Errorf("parse cron: %w", err)
	}
	parsed, ok := s.(*cron.SpecSchedule)
	if !ok {
		return nil, errCronMustContainFiveFields
	}
	parsed.Location = loc
	return s, nil
}
