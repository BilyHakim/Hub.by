package httpapi

import (
	"math"
	"testing"
)

func TestAnnuityPayment(t *testing.T) {
	got := annuityPayment(400_000_000, 5, 180)
	if math.Abs(got-3_163_174) > 2 {
		t.Fatalf("payment = %.0f, want approximately 3163174", got)
	}
}

func TestCalculateMortgageSwitchesToFloatingRate(t *testing.T) {
	result := calculateMortgage(mortgageInput{
		PropertyPrice: 500_000_000, DownPaymentPercent: 20,
		TenorYears: 15, FixedRate: 5, FixedYears: 5, FloatingRate: 10,
		MonthlyIncome: 20_000_000, StartDate: "2026-08-01",
	})
	schedule := result["schedule"].([]mortgageInstallment)
	if len(schedule) != 180 {
		t.Fatalf("schedule length = %d, want 180", len(schedule))
	}
	if schedule[59].RateType != "fixed" || schedule[60].RateType != "floating" {
		t.Fatalf("unexpected rate transition: %s to %s", schedule[59].RateType, schedule[60].RateType)
	}
	if schedule[len(schedule)-1].RemainingBalance != 0 {
		t.Fatalf("remaining balance = %d, want 0", schedule[len(schedule)-1].RemainingBalance)
	}
}

func TestMortgageHealth(t *testing.T) {
	tests := []struct {
		ratio float64
		want  string
	}{
		{29.9, "healthy"},
		{35, "safe"},
		{45, "burdened"},
		{51, "unhealthy"},
	}
	for _, test := range tests {
		got, _ := mortgageHealth(test.ratio)
		if got != test.want {
			t.Fatalf("status for %.1f = %s, want %s", test.ratio, got, test.want)
		}
	}
}
