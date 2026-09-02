package httpapi

import (
	"net/http"
	"time"
)

type financialHealthSummary struct {
	FirstTransaction      string                    `json:"firstTransaction"`
	LastTransaction       string                    `json:"lastTransaction"`
	TransactionCount      int64                     `json:"transactionCount"`
	TotalBalance          int64                     `json:"totalBalance"`
	TotalAssets           int64                     `json:"totalAssets"`
	TotalLiabilities      int64                     `json:"totalLiabilities"`
	NetWorth              int64                     `json:"netWorth"`
	LifetimeIncome        int64                     `json:"lifetimeIncome"`
	LifetimeExpense       int64                     `json:"lifetimeExpense"`
	LifetimeSavings       int64                     `json:"lifetimeSavings"`
	SavingsRate           float64                   `json:"savingsRate"`
	AverageMonthlyExpense int64                     `json:"averageMonthlyExpense"`
	ActiveMonths          int64                     `json:"activeMonths"`
	EmergencyFund         int64                     `json:"emergencyFund"`
	EmergencyTarget       int64                     `json:"emergencyTarget"`
	EmergencyProgress     float64                   `json:"emergencyProgress"`
	Accounts              []financialHealthAccount  `json:"accounts"`
	ExpenseCategories     []financialHealthCategory `json:"expenseCategories"`
}

type financialHealthAccount struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	Balance         int64   `json:"balance"`
	Share           float64 `json:"share"`
	IsEmergencyFund bool    `json:"isEmergencyFund"`
}

type financialHealthCategory struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Color              string  `json:"color"`
	Value              int64   `json:"value"`
	Share              float64 `json:"share"`
	TransactionCount   int64   `json:"transactionCount"`
	AverageTransaction int64   `json:"averageTransaction"`
	LargestAmount      int64   `json:"largestAmount"`
	LargestDescription string  `json:"largestDescription"`
	LastSpentAt        string  `json:"lastSpentAt"`
}

// financialHealth returns a period-free view over every transaction in the
// active workspace. Account balances remain point-in-time values by design.
func (api *API) financialHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceID, err := api.currentWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}

	result := financialHealthSummary{
		Accounts:          make([]financialHealthAccount, 0),
		ExpenseCategories: make([]financialHealthCategory, 0),
	}
	var firstDate, lastDate *time.Time
	err = api.db.QueryRow(ctx, `
		WITH transaction_summary AS (
			SELECT MIN(occurred_at) AS first_date, MAX(occurred_at) AS last_date,
				COUNT(*)::bigint AS transaction_count,
				COALESCE(SUM(amount) FILTER (WHERE type='income'), 0)::bigint AS income,
				COALESCE(SUM(amount) FILTER (WHERE type='expense'), 0)::bigint AS expense
			FROM transactions WHERE workspace_id=$1
		), account_summary AS (
			SELECT
				COALESCE(SUM(current_balance) FILTER (WHERE kind IN ('cash','bank','ewallet')), 0)::bigint AS balance,
				COALESCE(SUM(current_balance) FILTER (WHERE kind <> 'liability'), 0)::bigint AS assets,
				ABS(COALESCE(SUM(current_balance) FILTER (WHERE kind='liability'), 0))::bigint AS liabilities,
				COALESCE(SUM(current_balance) FILTER (WHERE is_emergency_fund), 0)::bigint AS emergency
			FROM accounts WHERE workspace_id=$1
		), emergency AS (
			SELECT COALESCE(AVG(monthly_expense) * MAX(target_months), 0)::bigint AS target
			FROM emergency_fund_settings WHERE workspace_id=$1
		)
		SELECT t.first_date, t.last_date, t.transaction_count, t.income, t.expense,
			a.balance, a.assets, a.liabilities, a.emergency, e.target
		FROM transaction_summary t, account_summary a, emergency e
	`, workspaceID).Scan(
		&firstDate, &lastDate, &result.TransactionCount, &result.LifetimeIncome,
		&result.LifetimeExpense, &result.TotalBalance, &result.TotalAssets,
		&result.TotalLiabilities, &result.EmergencyFund, &result.EmergencyTarget,
	)
	if err != nil {
		api.logger.Error("load financial health summary", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load financial health")
		return
	}

	result.NetWorth = result.TotalAssets - result.TotalLiabilities
	result.LifetimeSavings = result.LifetimeIncome - result.LifetimeExpense
	if result.LifetimeIncome > 0 {
		result.SavingsRate = float64(result.LifetimeSavings) / float64(result.LifetimeIncome) * 100
	}
	if result.EmergencyTarget > 0 {
		result.EmergencyProgress = float64(result.EmergencyFund) / float64(result.EmergencyTarget) * 100
	}
	if firstDate != nil && lastDate != nil {
		result.FirstTransaction = firstDate.Format("2006-01-02")
		result.LastTransaction = lastDate.Format("2006-01-02")
		result.ActiveMonths = int64((lastDate.Year()-firstDate.Year())*12 + int(lastDate.Month()-firstDate.Month()) + 1)
		if result.ActiveMonths > 0 {
			result.AverageMonthlyExpense = result.LifetimeExpense / result.ActiveMonths
		}
	}

	accountRows, err := api.db.Query(ctx, `
		SELECT id, name, kind::text, current_balance, is_emergency_fund
		FROM accounts WHERE workspace_id=$1 ORDER BY current_balance DESC, name
	`, workspaceID)
	if err == nil {
		defer accountRows.Close()
		for accountRows.Next() {
			var item financialHealthAccount
			if scanErr := accountRows.Scan(&item.ID, &item.Name, &item.Kind, &item.Balance, &item.IsEmergencyFund); scanErr == nil {
				if result.TotalAssets > 0 && item.Balance > 0 && item.Kind != "liability" {
					item.Share = float64(item.Balance) / float64(result.TotalAssets) * 100
				}
				result.Accounts = append(result.Accounts, item)
			}
		}
	}

	categoryRows, err := api.db.Query(ctx, `
		SELECT c.id, c.name, c.color, SUM(t.amount)::bigint, COUNT(*)::bigint,
			ROUND(AVG(t.amount))::bigint,
			(ARRAY_AGG(t.amount ORDER BY t.amount DESC, t.id DESC))[1]::bigint,
			(ARRAY_AGG(COALESCE(NULLIF(t.description,''), c.name) ORDER BY t.amount DESC, t.id DESC))[1],
			MAX(t.occurred_at)
		FROM transactions t
		JOIN categories c ON c.id=t.category_id
		WHERE t.workspace_id=$1 AND t.type='expense'
		GROUP BY c.id, c.name, c.color
		ORDER BY SUM(t.amount) DESC, c.name
	`, workspaceID)
	if err == nil {
		defer categoryRows.Close()
		for categoryRows.Next() {
			var item financialHealthCategory
			var lastSpent time.Time
			if scanErr := categoryRows.Scan(&item.ID, &item.Name, &item.Color, &item.Value,
				&item.TransactionCount, &item.AverageTransaction, &item.LargestAmount,
				&item.LargestDescription, &lastSpent); scanErr == nil {
				if result.LifetimeExpense > 0 {
					item.Share = float64(item.Value) / float64(result.LifetimeExpense) * 100
				}
				item.LastSpentAt = lastSpent.Format("2006-01-02")
				result.ExpenseCategories = append(result.ExpenseCategories, item)
			}
		}
	}

	writeJSON(w, http.StatusOK, envelope{"data": result})
}
