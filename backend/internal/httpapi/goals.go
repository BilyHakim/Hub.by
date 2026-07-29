package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type goalInput struct {
	Name                string  `json:"name"`
	TargetAmount        int64   `json:"targetAmount"`
	CurrentAmount       int64   `json:"currentAmount"`
	MonthlyContribution int64   `json:"monthlyContribution"`
	TargetDate          string  `json:"targetDate"`
	Icon                string  `json:"icon"`
	ExpectedReturn      float64 `json:"expectedReturn"`
}

func (api *API) listGoals(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT id,name,target_amount,current_amount,monthly_contribution,
		       target_date,icon,expected_return
		FROM financial_goals WHERE workspace_id=$1 ORDER BY target_date,id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load goals")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var input goalInput
		var targetDate time.Time
		if rows.Scan(
			&id, &input.Name, &input.TargetAmount, &input.CurrentAmount,
			&input.MonthlyContribution, &targetDate, &input.Icon, &input.ExpectedReturn,
		) == nil {
			input.TargetDate = targetDate.Format("2006-01-02")
			items = append(items, goalResponse(id, input, time.Now()))
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) createGoal(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input goalInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	targetDate, ok := validGoalInput(&input)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid financial goal values")
		return
	}
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO financial_goals(
			workspace_id,name,target_amount,current_amount,monthly_contribution,
			target_date,icon,expected_return
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, workspaceID, input.Name, input.TargetAmount, input.CurrentAmount,
		input.MonthlyContribution, targetDate, input.Icon, input.ExpectedReturn).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create financial goal")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": goalResponse(id, input, time.Now())})
}

func (api *API) replaceGoal(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid goal id")
		return
	}
	var input goalInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	targetDate, ok := validGoalInput(&input)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid financial goal values")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE financial_goals SET
			name=$1,target_amount=$2,current_amount=$3,monthly_contribution=$4,
			target_date=$5,icon=$6,expected_return=$7,updated_at=now()
		WHERE id=$8 AND workspace_id=$9
	`, input.Name, input.TargetAmount, input.CurrentAmount, input.MonthlyContribution,
		targetDate, input.Icon, input.ExpectedReturn, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update financial goal")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "goal not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": goalResponse(id, input, time.Now())})
}

func (api *API) updateGoal(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid goal id")
		return
	}
	var input struct {
		CurrentAmount int64 `json:"currentAmount"`
	}
	if decodeJSON(r, &input) != nil || input.CurrentAmount < 0 {
		writeError(w, http.StatusBadRequest, "currentAmount must be zero or greater")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE financial_goals SET current_amount=$1,updated_at=now()
		WHERE id=$2 AND workspace_id=$3
	`, input.CurrentAmount, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update goal")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "goal not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "currentAmount": input.CurrentAmount}})
}

func (api *API) deleteGoal(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid goal id")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		DELETE FROM financial_goals WHERE id=$1 AND workspace_id=$2
	`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete financial goal")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "goal not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validGoalInput(input *goalInput) (time.Time, bool) {
	input.Name = strings.TrimSpace(input.Name)
	input.Icon = strings.TrimSpace(input.Icon)
	if input.Icon == "" {
		input.Icon = "target"
	}
	targetDate, err := time.Parse("2006-01-02", input.TargetDate)
	valid := len(input.Name) >= 2 && len(input.Name) <= 100 &&
		input.TargetAmount > 0 && input.CurrentAmount >= 0 &&
		input.MonthlyContribution >= 0 &&
		input.ExpectedReturn >= 0 && input.ExpectedReturn <= 100 && err == nil
	return targetDate, valid
}

func goalResponse(id int64, input goalInput, now time.Time) envelope {
	targetDate, _ := time.Parse("2006-01-02", input.TargetDate)
	monthsRemaining := monthsUntil(now, targetDate)
	estimatedWithout := monthsToGoal(
		float64(input.CurrentAmount), float64(input.TargetAmount),
		float64(input.MonthlyContribution), 0,
	)
	estimatedWith := monthsToGoal(
		float64(input.CurrentAmount), float64(input.TargetAmount),
		float64(input.MonthlyContribution), input.ExpectedReturn/100/12,
	)
	projected := projectGoalValue(
		float64(input.CurrentAmount), float64(input.MonthlyContribution),
		input.ExpectedReturn/100/12, monthsRemaining,
	)
	required := requiredMonthlyContribution(
		float64(input.TargetAmount), float64(input.CurrentAmount),
		input.ExpectedReturn/100/12, monthsRemaining,
	)
	progress := ratio(input.CurrentAmount, input.TargetAmount)
	remaining := max(input.TargetAmount-input.CurrentAmount, 0)
	status := "on_track"
	if input.CurrentAmount >= input.TargetAmount {
		status = "completed"
	} else if !targetDate.After(now) {
		status = "overdue"
	} else if projected < float64(input.TargetAmount) {
		status = "at_risk"
	}
	return envelope{
		"id": id, "name": input.Name, "targetAmount": input.TargetAmount,
		"currentAmount": input.CurrentAmount, "monthlyContribution": input.MonthlyContribution,
		"targetDate": input.TargetDate, "icon": input.Icon, "expectedReturn": input.ExpectedReturn,
		"progress": progress, "remainingAmount": remaining, "monthsRemaining": monthsRemaining,
		"estimatedMonthsWithoutInvestment": estimatedWithout,
		"estimatedMonthsWithInvestment":    estimatedWith,
		"projectedValueAtTarget":           roundMoney(projected),
		"requiredMonthlyContribution":      roundMoney(required), "status": status,
	}
}

func monthsUntil(now, target time.Time) int {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	target = time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, today.Location())
	if !target.After(today) {
		return 0
	}
	months := (target.Year()-today.Year())*12 + int(target.Month()-today.Month())
	if target.Day() > today.Day() {
		months++
	}
	return max(months, 1)
}

func monthsToGoal(current, target, contribution, monthlyRate float64) int {
	if current >= target {
		return 0
	}
	if contribution <= 0 && monthlyRate <= 0 {
		return -1
	}
	balance := current
	for month := 1; month <= 1200; month++ {
		balance += balance*monthlyRate + contribution
		if balance >= target {
			return month
		}
	}
	return -1
}

func projectGoalValue(current, contribution, monthlyRate float64, months int) float64 {
	balance := current
	for range months {
		balance += balance*monthlyRate + contribution
	}
	return balance
}
