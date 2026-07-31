package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type budgetItem struct {
	CategoryID   int64  `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	Color        string `json:"color"`
	Icon         string `json:"icon"`
	Planned      int64  `json:"planned"`
	Actual       int64  `json:"actual"`
	Remaining    int64  `json:"remaining"`
}

type budgetSummary struct {
	Month       string       `json:"month"`
	PeriodStart string       `json:"periodStart"`
	PeriodEnd   string       `json:"periodEnd"`
	Planned     int64        `json:"planned"`
	Actual      int64        `json:"actual"`
	Remaining   int64        `json:"remaining"`
	Items       []budgetItem `json:"items"`
}

type budgetPlanInput struct {
	Items []budgetPlanItemInput `json:"items"`
}

type budgetPlanItemInput struct {
	CategoryID    int64 `json:"categoryId"`
	PlannedAmount int64 `json:"plannedAmount"`
}

func (api *API) getBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	month, monthDate, period, err := api.resolveBudgetPeriod(r, workspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "month must use YYYY-MM")
		return
	}

	rows, err := api.db.Query(r.Context(), `
		WITH actual AS (
			SELECT category_id, COALESCE(SUM(amount), 0)::bigint AS amount
			FROM transactions
			WHERE workspace_id=$1 AND type='expense'
			  AND occurred_at >= $3 AND occurred_at < $4
			GROUP BY category_id
		)
		SELECT c.id, c.name, c.color, c.icon,
		       COALESCE(b.planned_amount, 0)::bigint,
		       COALESCE(a.amount, 0)::bigint
		FROM categories c
		LEFT JOIN monthly_budgets b
		  ON b.workspace_id=c.workspace_id AND b.category_id=c.id AND b.month=$2
		LEFT JOIN actual a ON a.category_id=c.id
		WHERE c.workspace_id=$1 AND c.type='expense'
		ORDER BY c.id
	`, workspaceID, monthDate, period.Start, period.EndExclusive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load budget")
		return
	}
	defer rows.Close()

	result := budgetSummary{
		Month: month, PeriodStart: period.Start.Format("2006-01-02"),
		PeriodEnd: period.EndInclusive.Format("2006-01-02"), Items: make([]budgetItem, 0),
	}
	for rows.Next() {
		var item budgetItem
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.Color, &item.Icon, &item.Planned, &item.Actual); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read budget")
			return
		}
		item.Remaining = item.Planned - item.Actual
		result.Planned += item.Planned
		result.Actual += item.Actual
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read budget")
		return
	}
	result.Remaining = result.Planned - result.Actual
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (api *API) replaceBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	_, monthDate, _, err := api.resolveBudgetPeriod(r, workspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "month must use YYYY-MM")
		return
	}
	var input budgetPlanInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validBudgetItems(input.Items) {
		writeError(w, http.StatusUnprocessableEntity, "budget items must have unique expense categories and non-negative amounts")
		return
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save budget")
		return
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(r.Context(), `DELETE FROM monthly_budgets WHERE workspace_id=$1 AND month=$2`, workspaceID, monthDate); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save budget")
		return
	}
	for _, item := range input.Items {
		result, err := tx.Exec(r.Context(), `
			INSERT INTO monthly_budgets(workspace_id,category_id,month,planned_amount)
			SELECT $1,c.id,$3,$4
			FROM categories c
			WHERE c.id=$2 AND c.workspace_id=$1 AND c.type='expense'
		`, workspaceID, item.CategoryID, monthDate, item.PlannedAmount)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save budget")
			return
		}
		if result.RowsAffected() != 1 {
			writeError(w, http.StatusUnprocessableEntity, "budget category does not belong to the active workspace")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save budget")
		return
	}
	api.getBudget(w, r)
}

func (api *API) resolveBudgetPeriod(r *http.Request, workspaceID int64) (string, time.Time, financePeriod, error) {
	month := r.URL.Query().Get("month")
	if month == "" {
		setting, err := api.financePeriodSetting(r.Context(), workspaceID)
		if err != nil {
			return "", time.Time{}, financePeriod{}, err
		}
		month = currentFinancePeriodLabel(time.Now(), setting)
	}
	monthDate, err := time.Parse("2006-01-02", month+"-01")
	if err != nil || monthDate.Format("2006-01") != month {
		return "", time.Time{}, financePeriod{}, errInvalidBudgetMonth
	}
	period, err := api.resolveFinancePeriod(r.Context(), workspaceID, month)
	return month, monthDate, period, err
}

func validBudgetItems(items []budgetPlanItemInput) bool {
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.CategoryID <= 0 || item.PlannedAmount < 0 {
			return false
		}
		if _, exists := seen[item.CategoryID]; exists {
			return false
		}
		seen[item.CategoryID] = struct{}{}
	}
	return true
}

var errInvalidBudgetMonth = errors.New("invalid budget month")
