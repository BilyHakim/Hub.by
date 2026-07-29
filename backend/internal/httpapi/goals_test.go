package httpapi

import (
	"testing"
	"time"
)

func TestMonthsToGoalMatchesWorkbook(t *testing.T) {
	months := monthsToGoal(5_000_000, 30_000_000, 1_000_000, 0.05/12)
	if months != 24 {
		t.Fatalf("months with investment = %d, want 24", months)
	}
	withoutInvestment := monthsToGoal(5_000_000, 30_000_000, 1_000_000, 0)
	if withoutInvestment != 25 {
		t.Fatalf("months without investment = %d, want 25", withoutInvestment)
	}
}

func TestGoalResponseStatus(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.Local)
	input := goalInput{
		Name: "DP Rumah", TargetAmount: 100_000_000, CurrentAmount: 10_000_000,
		MonthlyContribution: 1_000_000, TargetDate: "2027-07-29",
		Icon: "home", ExpectedReturn: 5,
	}
	result := goalResponse(1, input, now)
	if result["status"] != "at_risk" {
		t.Fatalf("status = %v, want at_risk", result["status"])
	}
	if result["monthsRemaining"] != 12 {
		t.Fatalf("months remaining = %v, want 12", result["monthsRemaining"])
	}
}
