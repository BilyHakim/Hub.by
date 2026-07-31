package httpapi

import "testing"

func TestValidBudgetItems(t *testing.T) {
	tests := []struct {
		name  string
		items []budgetPlanItemInput
		want  bool
	}{
		{name: "empty budget", items: nil, want: true},
		{name: "valid items", items: []budgetPlanItemInput{{CategoryID: 1, PlannedAmount: 500_000}, {CategoryID: 2}}, want: true},
		{name: "invalid category", items: []budgetPlanItemInput{{CategoryID: 0, PlannedAmount: 500_000}}, want: false},
		{name: "negative amount", items: []budgetPlanItemInput{{CategoryID: 1, PlannedAmount: -1}}, want: false},
		{name: "duplicate category", items: []budgetPlanItemInput{{CategoryID: 1}, {CategoryID: 1}}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validBudgetItems(test.items); got != test.want {
				t.Fatalf("validBudgetItems() = %t, want %t", got, test.want)
			}
		})
	}
}
