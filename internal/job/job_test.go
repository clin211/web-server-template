package job

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRegistryValidatesVersionsAndFreezes(t *testing.T) {
	r := NewRegistry()
	h := func(context.Context, ReportParams) (Result, error) { return Result{}, nil }
	if err := Register(r, "report.generate", 1, ValidateReport, h); err != nil {
		t.Fatal(err)
	}
	if err := Register(r, "report.generate", 1, ValidateReport, h); err == nil {
		t.Fatal("duplicate accepted")
	}
	r.Freeze()
	if err := Register(r, "other", 1, ValidateReport, h); err == nil {
		t.Fatal("registration after freeze accepted")
	}
	for _, tc := range []struct {
		version int
		payload string
		ok      bool
	}{{1, `{"days":7}`, true}, {1, `{"days":0}`, false}, {1, `{"days":7,"password":"secret"}`, false}, {2, `{"days":7}`, false}} {
		err := r.Validate("report.generate", tc.version, []byte(tc.payload))
		if (err == nil) != tc.ok {
			t.Errorf("%+v: %v", tc, err)
		}
	}
}
func TestCronDSTAndValidation(t *testing.T) {
	for _, tc := range []struct {
		cron, zone string
		ok         bool
	}{{"0 2 * * *", "America/New_York", true}, {"* * * * * *", "UTC", false}, {"CRON_TZ=UTC * * * * *", "UTC", false}, {"* * * * *", "wrong", false}} {
		_, err := ParseSchedule(tc.cron, tc.zone)
		if (err == nil) != tc.ok {
			t.Errorf("%+v %v", tc, err)
		}
	}
	s, err := ParseSchedule("30 2 * * *", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	next := s.Next(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
	if next.Day() != 9 {
		t.Fatalf("DST missing time: %s", next)
	}
	s, err = ParseSchedule("30 1 * * *", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	a := s.Next(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	b := s.Next(a)
	if b.Sub(a) != time.Hour {
		t.Fatalf("DST duplicate: %s %s", a, b)
	}
}
func TestPermanentAndBackoff(t *testing.T) {
	if !IsPermanent(Permanent(errors.New("invalid"))) {
		t.Fatal("classification")
	}
	for n := 1; n < 100; n++ {
		d := RetryDelay(n)
		if d < 0 || d > 5*time.Minute {
			t.Fatalf("delay %s", d)
		}
	}
}
