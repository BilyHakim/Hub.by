package httpapi

import (
	"net/http"
	"time"
)

type dashboardSummary struct {
	Month             string          `json:"month"`
	PeriodStart       string          `json:"periodStart"`
	PeriodEnd         string          `json:"periodEnd"`
	Income            int64           `json:"income"`
	Expense           int64           `json:"expense"`
	Savings           int64           `json:"savings"`
	SavingsRate       float64         `json:"savingsRate"`
	NetWorth          int64           `json:"netWorth"`
	EmergencyFund     int64           `json:"emergencyFund"`
	EmergencyTarget   int64           `json:"emergencyTarget"`
	EmergencyProgress float64         `json:"emergencyProgress"`
	InvestmentValue   int64           `json:"investmentValue"`
	InvestmentReturn  float64         `json:"investmentReturn"`
	Cashflow          []cashflowPoint `json:"cashflow"`
	ExpenseBreakdown  []categoryPoint `json:"expenseBreakdown"`
	FinancialCheckup  []checkupItem   `json:"financialCheckup"`
}

type cashflowPoint struct {
	Month   string `json:"month"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

type categoryPoint struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
	Color string `json:"color"`
}

type checkupItem struct {
	Label          string  `json:"label"`
	Value          float64 `json:"value"`
	Recommendation string  `json:"recommendation"`
	Status         string  `json:"status"`
}

func (api *API) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceID, err := api.currentWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	period, err := api.resolveFinancePeriod(ctx, workspaceID, month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid finance period")
		return
	}

	var result dashboardSummary
	result.Month = month
	result.PeriodStart = period.Start.Format("2006-01-02")
	result.PeriodEnd = period.EndInclusive.Format("2006-01-02")
	result.Cashflow = make([]cashflowPoint, 0)
	result.ExpenseBreakdown = make([]categoryPoint, 0)
	err = api.db.QueryRow(ctx, `
		WITH monthly AS (
			SELECT
				COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS income,
				COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS expense,
				COALESCE(SUM(amount) FILTER (WHERE type = 'expense' AND is_debt_payment), 0)::bigint AS debt
			FROM transactions
			WHERE occurred_at >= $1 AND occurred_at < $2 AND workspace_id = $3
		), wealth AS (
			SELECT
				COALESCE(SUM(current_balance) FILTER (WHERE kind <> 'liability'), 0)::bigint AS assets,
				ABS(COALESCE(SUM(current_balance) FILTER (WHERE kind = 'liability'), 0))::bigint AS liabilities,
				COALESCE(SUM(current_balance) FILTER (WHERE is_emergency_fund), 0)::bigint AS emergency
			FROM accounts WHERE workspace_id = $3
		), investments AS (
			SELECT COALESCE(SUM(current_value), 0)::bigint AS value,
			       COALESCE(SUM(purchase_value), 0)::bigint AS cost
			FROM investments WHERE workspace_id = $3
		), emergency AS (
			SELECT COALESCE(AVG(monthly_expense) * MAX(target_months), 0)::bigint AS target
			FROM emergency_fund_settings WHERE workspace_id = $3
		)
		SELECT monthly.income, monthly.expense,
		       wealth.assets - wealth.liabilities, wealth.emergency,
		       emergency.target, investments.value,
		       CASE WHEN investments.cost > 0 THEN ((investments.value - investments.cost)::numeric / investments.cost * 100) ELSE 0 END,
		       monthly.debt
		FROM monthly, wealth, investments, emergency
	`, period.Start, period.EndExclusive, workspaceID).Scan(
		&result.Income, &result.Expense, &result.NetWorth, &result.EmergencyFund,
		&result.EmergencyTarget, &result.InvestmentValue, &result.InvestmentReturn,
		new(int64),
	)
	if err != nil {
		api.logger.Error("load dashboard summary", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	result.Savings = result.Income - result.Expense
	if result.Income > 0 {
		result.SavingsRate = float64(result.Savings) / float64(result.Income) * 100
	}
	if result.EmergencyTarget > 0 {
		result.EmergencyProgress = float64(result.EmergencyFund) / float64(result.EmergencyTarget) * 100
	}

	labelMonth, _ := time.Parse("2006-01-02", month+"-01")
	monthLabels := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	for offset := -5; offset <= 0; offset++ {
		pointMonth := labelMonth.AddDate(0, offset, 0)
		pointPeriod, periodErr := buildFinancePeriod(pointMonth.Format("2006-01"), financePeriodSetting{
			Mode: period.Mode, StartDay: period.StartDay,
		})
		if periodErr != nil {
			continue
		}
		point := cashflowPoint{Month: monthLabels[int(pointMonth.Month())-1]}
		queryErr := api.db.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(amount) FILTER (WHERE type='income'),0)::bigint,
				COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)::bigint
			FROM transactions
			WHERE workspace_id=$1 AND occurred_at >= $2 AND occurred_at < $3
		`, workspaceID, pointPeriod.Start, pointPeriod.EndExclusive).Scan(&point.Income, &point.Expense)
		if queryErr == nil {
			result.Cashflow = append(result.Cashflow, point)
		}
	}

	colors := []string{"#49685c", "#e8a65d", "#d77268", "#7894a0", "#9a8bb7", "#b4a464"}
	rows, err := api.db.Query(ctx, `
		SELECT c.name, SUM(t.amount)::bigint
		FROM transactions t JOIN categories c ON c.id = t.category_id
		WHERE t.type = 'expense' AND t.occurred_at >= $1 AND t.occurred_at < $2
			AND t.workspace_id = $3
		GROUP BY c.name ORDER BY SUM(t.amount) DESC LIMIT 6
	`, period.Start, period.EndExclusive, workspaceID)
	if err == nil {
		defer rows.Close()
		index := 0
		for rows.Next() {
			var point categoryPoint
			if err := rows.Scan(&point.Name, &point.Value); err == nil {
				point.Color = colors[index%len(colors)]
				result.ExpenseBreakdown = append(result.ExpenseBreakdown, point)
				index++
			}
		}
	}

	debtRatio := 0.0
	if result.Income > 0 {
		var debt int64
		_ = api.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::bigint FROM transactions WHERE type='expense' AND is_debt_payment AND occurred_at >= $1 AND occurred_at < $2 AND workspace_id=$3`, period.Start, period.EndExclusive, workspaceID).Scan(&debt)
		debtRatio = float64(debt) / float64(result.Income) * 100
	}
	result.FinancialCheckup = []checkupItem{
		{Label: "Rasio tabungan", Value: result.SavingsRate, Recommendation: "Minimal 20%", Status: status(result.SavingsRate >= 20)},
		{Label: "Kewajiban / pendapatan", Value: debtRatio, Recommendation: "Maksimal 30%", Status: status(debtRatio <= 30)},
		{Label: "Dana darurat", Value: result.EmergencyProgress, Recommendation: "Minimal 100%", Status: status(result.EmergencyProgress >= 100)},
	}

	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func status(ok bool) string {
	if ok {
		return "healthy"
	}
	return "attention"
}
