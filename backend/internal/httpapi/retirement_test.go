package httpapi

import (
	"math"
	"testing"
)

func TestCalculateRetirementMatchesWorkbookFormula(t *testing.T) {
	result := calculateRetirement(retirementInput{
		CurrentAge: 25, RetirementAge: 55, MonthlyExpense: 5_000_000,
		InflationRate: 3, ExpectedReturn: 6, CurrentFund: 100_000_000,
		MonthlyContribution: 4_000_000, WithdrawalRate: 4,
	})
	summary := result["summary"].(envelope)
	expense := summary["monthlyExpenseAtRetirement"].(int64)
	if math.Abs(float64(expense)-12_136_312) > 2 {
		t.Fatalf("retirement expense = %d, want approximately 12136312", expense)
	}
	target := summary["targetFund"].(int64)
	if math.Abs(float64(target)-3_640_893_707) > 2 {
		t.Fatalf("target fund = %d, want approximately 3640893707", target)
	}
	projection := result["projection"].([]retirementPoint)
	if len(projection) != 30 || projection[len(projection)-1].Age != 55 {
		t.Fatalf("unexpected projection: length=%d final age=%d", len(projection), projection[len(projection)-1].Age)
	}
}

func TestRequiredMonthlyContributionWithoutReturn(t *testing.T) {
	got := requiredMonthlyContribution(220, 100, 0, 12)
	if got != 10 {
		t.Fatalf("contribution = %.2f, want 10", got)
	}
}
