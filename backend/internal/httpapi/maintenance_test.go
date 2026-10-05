package httpapi

import (
	"testing"
	"time"
)

func TestMaintenanceNextDate(t *testing.T) {
	cases := []struct {
		base       string
		interval   int
		unit, want string
	}{
		{"2026-01-31", 1, "months", "2026-02-28"},
		{"2024-01-31", 1, "months", "2024-02-29"},
		{"2024-02-29", 1, "years", "2025-02-28"},
		{"2026-01-10", 3, "months", "2026-04-10"},
		{"2026-12-31", 1, "days", "2027-01-01"},
		{"2026-01-10", 2, "weeks", "2026-01-24"},
	}
	for _, c := range cases {
		base, _ := time.Parse("2006-01-02", c.base)
		if got := maintenanceNextDate(base, c.interval, c.unit).Format("2006-01-02"); got != c.want {
			t.Errorf("%s + %d %s = %s, want %s", c.base, c.interval, c.unit, got, c.want)
		}
	}
}

func TestMaintenanceStatusAndLatestCompletion(t *testing.T) {
	interval := 3
	today, _ := time.Parse("2006-01-02", "2026-04-10")
	for _, c := range []struct{ date, status string }{{"2026-01-09", "overdue"}, {"2026-01-10", "due_today"}, {"2026-01-15", "due_soon"}, {"2026-02-10", "good"}} {
		rule := maintenanceRule{Active: true, StartDate: c.date, IntervalValue: &interval, IntervalUnit: "months", ReminderDays: 14}
		calculateMaintenanceRule(&rule, nil, today)
		if rule.Status != c.status {
			t.Errorf("start %s: got %s want %s", c.date, rule.Status, c.status)
		}
	}
	rule := maintenanceRule{Active: true, StartDate: "2026-01-10", LastMaintenance: "2026-04-01", IntervalValue: &interval, IntervalUnit: "months"}
	calculateMaintenanceRule(&rule, nil, today)
	if rule.NextDueDate != "2026-07-01" {
		t.Fatalf("next date %s", rule.NextDueDate)
	}
	rule.Active = false
	calculateMaintenanceRule(&rule, nil, today)
	if rule.Status != "inactive" {
		t.Fatal("inactive rule should not be due")
	}
}

func TestMaintenanceCombinedUsageStatus(t *testing.T) {
	interval := 3
	usageInterval := 2000.0
	lastUsage := 5000.0
	today, _ := time.Parse("2006-01-02", "2026-04-10")
	unknown := maintenanceRule{Active: true, StartDate: "2026-04-01", UsageInterval: &usageInterval}
	calculateMaintenanceRule(&unknown, nil, today)
	if unknown.Status != "usage_unknown" {
		t.Fatal("unknown usage must not be reported as good")
	}
	for _, c := range []struct {
		usage  float64
		status string
	}{{6000, "good"}, {6650, "due_soon"}, {7000, "due_today"}, {7001, "overdue"}} {
		rule := maintenanceRule{Active: true, StartDate: "2026-04-01", IntervalValue: &interval, IntervalUnit: "months", UsageInterval: &usageInterval, LastUsage: &lastUsage, ReminderUsage: 350}
		calculateMaintenanceRule(&rule, &c.usage, today)
		if rule.Status != c.status || *rule.NextDueUsage != 7000 {
			t.Errorf("usage %v: %s, due %v", c.usage, rule.Status, rule.NextDueUsage)
		}
	}
	rule := maintenanceRule{Active: true, StartDate: "2026-01-01", IntervalValue: &interval, IntervalUnit: "months", UsageInterval: &usageInterval}
	usage := 0.0
	calculateMaintenanceRule(&rule, &usage, today)
	if rule.Status != "overdue" {
		t.Fatal("usage must not override overdue time")
	}
}

func TestMaintenanceValidation(t *testing.T) {
	item := maintenanceItem{Name: "AC"}
	if !validMaintenanceItem(&item) {
		t.Fatal("optional fields must be optional")
	}
	item.ImageURL = "javascript:alert(1)"
	if validMaintenanceItem(&item) {
		t.Fatal("unsafe image URL accepted")
	}
	item.ImageURL = ""
	item.PurchaseDate = "2026-02-30"
	if validMaintenanceItem(&item) {
		t.Fatal("invalid date accepted")
	}
	rule := maintenanceRule{Name: "Clean", Kind: "maintenance", StartDate: "2026-01-01"}
	if validMaintenanceRule(&rule) {
		t.Fatal("missing interval accepted")
	}
	usage := 2000.0
	rule.UsageInterval = &usage
	if !validMaintenanceRule(&rule) {
		t.Fatal("usage-only rule rejected")
	}
	zero := 0
	rule.IntervalValue = &zero
	rule.IntervalUnit = "months"
	if validMaintenanceRule(&rule) {
		t.Fatal("zero interval accepted")
	}
}
