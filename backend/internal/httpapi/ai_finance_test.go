package httpapi

import (
	"context"
	"testing"
)

func TestAIPeriodRange(t *testing.T) {
	for _, tc := range []struct {
		start, end, first, last string
		count                   int
	}{
		{"2026-11", "2026-11", "2026-11", "2026-11", 1},
		{"2025-12", "2026-02", "2026-02", "2025-12", 3},
		{"2025-12", "2026-11", "2026-11", "2025-12", 12},
		{"", "2026-02", "2026-02", "2025-11", 4},
		{"", "1900-01", "1900-01", "1900-01", 1},
	} {
		labels, err := aiPeriodRange(tc.start, tc.end)
		if err != nil || len(labels) != tc.count || labels[0] != tc.first || labels[len(labels)-1] != tc.last {
			t.Fatalf("range %s to %s: %v, %v", tc.start, tc.end, labels, err)
		}
		for _, setting := range []financePeriodSetting{{Mode: periodModeFixedDay, StartDay: 29}, {Mode: periodModeEndOfMonth}} {
			for i := 1; i < len(labels); i++ {
				later, _ := buildFinancePeriod(labels[i-1], setting)
				earlier, _ := buildFinancePeriod(labels[i], setting)
				if !earlier.EndExclusive.Equal(later.Start) {
					t.Fatal("Finance periods overlap or leave a gap", labels, setting)
				}
			}
		}
	}
	for _, pair := range [][2]string{{"2026-11", "2026-10"}, {"2025-11", "2026-11"}, {"2026-00", "2026-11"}, {"2026-01", "2026-13"}, {"1899-12", "1900-01"}, {"2200-12", "2201-01"}, {"2026-1", "2026-11"}, {"2026-11", ""}} {
		if _, err := aiPeriodRange(pair[0], pair[1]); err == nil {
			t.Fatal("invalid range accepted", pair)
		}
	}
}

func TestAIRangeWithoutFinanceDoesNotReadDatabase(t *testing.T) {
	api := &API{}
	snapshot, err := api.aiFinanceContext(context.Background(), 1, aiSettings{}, "2026-09", "2026-11", 10)
	if err != nil || snapshot.StartMonth != "2026-09" || snapshot.Month != "2026-11" || len(snapshot.Periods) != 0 {
		t.Fatalf("unexpected disabled Finance context: %+v %v", snapshot, err)
	}
	if _, err := api.aiFinanceContext(context.Background(), 1, aiSettings{}, "2026-11", "2026-09", 10); err == nil {
		t.Fatal("disabled Finance bypassed range validation")
	}
}
