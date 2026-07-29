package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	periodModeFixedDay   = "fixed_day"
	periodModeEndOfMonth = "end_of_month"
)

type financePeriodSetting struct {
	Mode     string
	StartDay int
}

type financePeriod struct {
	Label        string    `json:"label"`
	Start        time.Time `json:"start"`
	EndExclusive time.Time `json:"-"`
	EndInclusive time.Time `json:"end"`
	StartDay     int       `json:"startDay"`
	Mode         string    `json:"mode"`
}

func (api *API) getFinancePeriodSetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	setting, err := api.financePeriodSetting(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load finance period setting")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = currentFinancePeriodLabel(time.Now(), setting)
	}
	period, err := buildFinancePeriod(month, setting)
	if err != nil {
		writeError(w, http.StatusBadRequest, "month must use YYYY-MM")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"periodMode":         setting.Mode,
		"periodStartDay":     setting.StartDay,
		"currentPeriodLabel": currentFinancePeriodLabel(time.Now(), setting),
		"example":            period,
	}})
}

func (api *API) updateFinancePeriodSetting(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input struct {
		PeriodMode     string `json:"periodMode"`
		PeriodStartDay int    `json:"periodStartDay"`
	}
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.PeriodMode == "" {
		input.PeriodMode = periodModeFixedDay
	}
	if input.PeriodMode != periodModeFixedDay && input.PeriodMode != periodModeEndOfMonth {
		writeError(w, http.StatusUnprocessableEntity, "periodMode must be fixed_day or end_of_month")
		return
	}
	if input.PeriodStartDay < 1 || input.PeriodStartDay > 31 {
		writeError(w, http.StatusUnprocessableEntity, "periodStartDay must be between 1 and 31")
		return
	}
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO workspace_finance_settings(workspace_id,period_start_day,period_mode)
		VALUES($1,$2,$3)
		ON CONFLICT(workspace_id) DO UPDATE
		SET period_start_day=EXCLUDED.period_start_day,
		    period_mode=EXCLUDED.period_mode,
		    updated_at=now()
	`, workspaceID, input.PeriodStartDay, input.PeriodMode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update finance period setting")
		return
	}
	setting := financePeriodSetting{Mode: input.PeriodMode, StartDay: input.PeriodStartDay}
	currentLabel := currentFinancePeriodLabel(time.Now(), setting)
	period, _ := buildFinancePeriod(currentLabel, setting)
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"periodMode":         setting.Mode,
		"periodStartDay":     input.PeriodStartDay,
		"currentPeriodLabel": currentLabel,
		"example":            period,
	}})
}

func (api *API) financePeriodSetting(ctx context.Context, workspaceID int64) (financePeriodSetting, error) {
	var setting financePeriodSetting
	err := api.db.QueryRow(ctx, `
		SELECT period_mode,period_start_day
		FROM workspace_finance_settings WHERE workspace_id=$1
	`, workspaceID).Scan(&setting.Mode, &setting.StartDay)
	if err == nil {
		return setting, nil
	}
	_, insertErr := api.db.Exec(ctx, `
		INSERT INTO workspace_finance_settings(workspace_id,period_start_day,period_mode)
		VALUES($1,1,$2) ON CONFLICT(workspace_id) DO NOTHING
	`, workspaceID, periodModeFixedDay)
	if insertErr != nil {
		return financePeriodSetting{}, insertErr
	}
	return financePeriodSetting{Mode: periodModeFixedDay, StartDay: 1}, nil
}

func normalizeFinancePeriodSetting(setting financePeriodSetting) financePeriodSetting {
	if setting.Mode != periodModeEndOfMonth {
		setting.Mode = periodModeFixedDay
	}
	if setting.StartDay < 1 || setting.StartDay > 31 {
		setting.StartDay = 1
	}
	return setting
}

func periodBoundary(month time.Time, setting financePeriodSetting) time.Time {
	setting = normalizeFinancePeriodSetting(setting)
	if setting.Mode == periodModeEndOfMonth {
		return time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.Local)
	}
	return clampedDate(month.Year(), month.Month(), setting.StartDay)
}

func (api *API) resolveFinancePeriod(ctx context.Context, workspaceID int64, month string) (financePeriod, error) {
	setting, err := api.financePeriodSetting(ctx, workspaceID)
	if err != nil {
		return financePeriod{}, err
	}
	return buildFinancePeriod(month, setting)
}

func buildFinancePeriod(month string, setting financePeriodSetting) (financePeriod, error) {
	setting = normalizeFinancePeriodSetting(setting)
	labelMonth, err := time.Parse("2006-01-02", month+"-01")
	if err != nil {
		return financePeriod{}, fmt.Errorf("parse period month: %w", err)
	}
	var startMonth, endMonth time.Time
	if setting.Mode == periodModeFixedDay && setting.StartDay == 1 {
		startMonth = labelMonth
		endMonth = labelMonth.AddDate(0, 1, 0)
	} else {
		startMonth = labelMonth.AddDate(0, -1, 0)
		endMonth = labelMonth
	}
	start := periodBoundary(startMonth, setting)
	endExclusive := periodBoundary(endMonth, setting)
	if !endExclusive.After(start) {
		endExclusive = start.AddDate(0, 1, 0)
	}
	return financePeriod{
		Label: month, Start: start, EndExclusive: endExclusive,
		EndInclusive: endExclusive.AddDate(0, 0, -1), StartDay: setting.StartDay,
		Mode: setting.Mode,
	}, nil
}

func clampedDate(year int, month time.Month, day int) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func currentFinancePeriodLabel(now time.Time, setting financePeriodSetting) string {
	setting = normalizeFinancePeriodSetting(setting)
	if setting.Mode == periodModeFixedDay && setting.StartDay == 1 {
		return now.Format("2006-01")
	}
	boundary := periodBoundary(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), setting)
	if !now.Before(boundary) {
		return now.AddDate(0, 1, 0).Format("2006-01")
	}
	return now.Format("2006-01")
}
