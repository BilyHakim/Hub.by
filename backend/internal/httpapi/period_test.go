package httpapi

import (
	"testing"
	"time"
)

func TestBuildFinancePeriod(t *testing.T) {
	tests := []struct {
		name, month string
		setting     financePeriodSetting
		wantStart   string
		wantEnd     string
	}{
		{"calendar month", "2026-08", financePeriodSetting{Mode: periodModeFixedDay, StartDay: 1}, "2026-08-01", "2026-08-31"},
		{"salary on 26th", "2026-08", financePeriodSetting{Mode: periodModeFixedDay, StartDay: 26}, "2026-07-26", "2026-08-25"},
		{"salary on 31st", "2026-08", financePeriodSetting{Mode: periodModeFixedDay, StartDay: 31}, "2026-07-31", "2026-08-30"},
		{"short February", "2026-03", financePeriodSetting{Mode: periodModeFixedDay, StartDay: 31}, "2026-02-28", "2026-03-30"},
		{"leap February", "2028-03", financePeriodSetting{Mode: periodModeFixedDay, StartDay: 31}, "2028-02-29", "2028-03-30"},
		{"end of month August", "2026-08", financePeriodSetting{Mode: periodModeEndOfMonth, StartDay: 31}, "2026-07-31", "2026-08-30"},
		{"end of month October", "2026-10", financePeriodSetting{Mode: periodModeEndOfMonth, StartDay: 31}, "2026-09-30", "2026-10-30"},
		{"end of month March leap year", "2028-03", financePeriodSetting{Mode: periodModeEndOfMonth, StartDay: 31}, "2028-02-29", "2028-03-30"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			period, err := buildFinancePeriod(test.month, test.setting)
			if err != nil {
				t.Fatal(err)
			}
			if got := period.Start.Format("2006-01-02"); got != test.wantStart {
				t.Fatalf("start = %s, want %s", got, test.wantStart)
			}
			if got := period.EndInclusive.Format("2006-01-02"); got != test.wantEnd {
				t.Fatalf("end = %s, want %s", got, test.wantEnd)
			}
		})
	}
}

func TestCurrentFinancePeriodLabel(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.Local)
	if got := currentFinancePeriodLabel(now, financePeriodSetting{Mode: periodModeFixedDay, StartDay: 26}); got != "2026-08" {
		t.Fatalf("label = %s, want 2026-08", got)
	}
	if got := currentFinancePeriodLabel(now, financePeriodSetting{Mode: periodModeFixedDay, StartDay: 31}); got != "2026-07" {
		t.Fatalf("label = %s, want 2026-07", got)
	}
	endOfMonth := financePeriodSetting{Mode: periodModeEndOfMonth, StartDay: 31}
	if got := currentFinancePeriodLabel(time.Date(2026, time.August, 31, 12, 0, 0, 0, time.Local), endOfMonth); got != "2026-09" {
		t.Fatalf("label at August month end = %s, want 2026-09", got)
	}
	if got := currentFinancePeriodLabel(time.Date(2026, time.September, 29, 12, 0, 0, 0, time.Local), endOfMonth); got != "2026-09" {
		t.Fatalf("label before month end = %s, want 2026-09", got)
	}
	if got := currentFinancePeriodLabel(time.Date(2026, time.September, 30, 12, 0, 0, 0, time.Local), endOfMonth); got != "2026-10" {
		t.Fatalf("label at month end = %s, want 2026-10", got)
	}
}
