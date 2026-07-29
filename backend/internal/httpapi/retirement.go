package httpapi

import (
	"math"
	"net/http"
)

type retirementInput struct {
	CurrentAge          int     `json:"currentAge"`
	RetirementAge       int     `json:"retirementAge"`
	MonthlyExpense      int64   `json:"monthlyExpense"`
	InflationRate       float64 `json:"inflationRate"`
	ExpectedReturn      float64 `json:"expectedReturn"`
	CurrentFund         int64   `json:"currentFund"`
	MonthlyContribution int64   `json:"monthlyContribution"`
	WithdrawalRate      float64 `json:"withdrawalRate"`
}

type retirementPoint struct {
	Year         int   `json:"year"`
	Age          int   `json:"age"`
	StartingFund int64 `json:"startingFund"`
	ReturnAmount int64 `json:"returnAmount"`
	Contribution int64 `json:"contribution"`
	EndingFund   int64 `json:"endingFund"`
}

func (api *API) getRetirementPlan(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO retirement_settings(
			workspace_id,current_age,retirement_age,monthly_expense,inflation_rate,
			expected_return,current_fund,monthly_contribution,withdrawal_rate
		) VALUES($1,25,55,5000000,3,6,100000000,4000000,4)
		ON CONFLICT(workspace_id) DO NOTHING
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare retirement plan")
		return
	}
	input, err := api.loadRetirementInput(r, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load retirement plan")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": calculateRetirement(input)})
}

func (api *API) updateRetirementPlan(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input retirementInput
	if decodeJSON(r, &input) != nil || !validRetirementInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid retirement planning values")
		return
	}
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO retirement_settings(
			workspace_id,current_age,retirement_age,monthly_expense,inflation_rate,
			expected_return,current_fund,monthly_contribution,withdrawal_rate
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(workspace_id) DO UPDATE SET
			current_age=EXCLUDED.current_age,
			retirement_age=EXCLUDED.retirement_age,
			monthly_expense=EXCLUDED.monthly_expense,
			inflation_rate=EXCLUDED.inflation_rate,
			expected_return=EXCLUDED.expected_return,
			current_fund=EXCLUDED.current_fund,
			monthly_contribution=EXCLUDED.monthly_contribution,
			withdrawal_rate=EXCLUDED.withdrawal_rate
	`, workspaceID, input.CurrentAge, input.RetirementAge, input.MonthlyExpense,
		input.InflationRate, input.ExpectedReturn, input.CurrentFund,
		input.MonthlyContribution, input.WithdrawalRate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save retirement plan")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": calculateRetirement(input)})
}

func (api *API) loadRetirementInput(r *http.Request, workspaceID int64) (retirementInput, error) {
	var input retirementInput
	err := api.db.QueryRow(r.Context(), `
		SELECT current_age,retirement_age,monthly_expense,inflation_rate,
		       expected_return,current_fund,monthly_contribution,withdrawal_rate
		FROM retirement_settings WHERE workspace_id=$1
	`, workspaceID).Scan(
		&input.CurrentAge, &input.RetirementAge, &input.MonthlyExpense,
		&input.InflationRate, &input.ExpectedReturn, &input.CurrentFund,
		&input.MonthlyContribution, &input.WithdrawalRate,
	)
	return input, err
}

func validRetirementInput(input retirementInput) bool {
	return input.CurrentAge >= 15 && input.CurrentAge <= 99 &&
		input.RetirementAge > input.CurrentAge && input.RetirementAge <= 100 &&
		input.MonthlyExpense >= 0 && input.CurrentFund >= 0 && input.MonthlyContribution >= 0 &&
		input.InflationRate >= 0 && input.InflationRate <= 100 &&
		input.ExpectedReturn >= 0 && input.ExpectedReturn <= 100 &&
		input.WithdrawalRate > 0 && input.WithdrawalRate <= 100
}

func calculateRetirement(input retirementInput) envelope {
	years := input.RetirementAge - input.CurrentAge
	months := years * 12
	expenseAtRetirement := float64(input.MonthlyExpense) *
		math.Pow(1+input.InflationRate/100, float64(years))
	annualExpense := expenseAtRetirement * 12
	targetFund := annualExpense / (input.WithdrawalRate / 100)
	balance := float64(input.CurrentFund)
	monthlyRate := input.ExpectedReturn / 100 / 12
	points := make([]retirementPoint, 0, years)
	yearStart := balance
	yearReturn := 0.0

	for month := 1; month <= months; month++ {
		investmentReturn := balance * monthlyRate
		balance += investmentReturn + float64(input.MonthlyContribution)
		yearReturn += investmentReturn
		if month%12 == 0 {
			points = append(points, retirementPoint{
				Year: month / 12, Age: input.CurrentAge + month/12,
				StartingFund: roundMoney(yearStart), ReturnAmount: roundMoney(yearReturn),
				Contribution: input.MonthlyContribution * 12, EndingFund: roundMoney(balance),
			})
			yearStart = balance
			yearReturn = 0
		}
	}
	projectedFund := balance
	gap := projectedFund - targetFund
	progress := 0.0
	if targetFund > 0 {
		progress = projectedFund / targetFund * 100
	}
	requiredContribution := requiredMonthlyContribution(
		targetFund, float64(input.CurrentFund), monthlyRate, months,
	)
	status := "on_track"
	message := "Proyeksi dana mencukupi target pensiun. Pertahankan setoran dan evaluasi asumsi secara berkala."
	if gap < 0 {
		status = "shortfall"
		message = "Proyeksi dana masih kurang. Tingkatkan setoran, evaluasi usia pensiun, atau sesuaikan target pengeluaran."
	}
	return envelope{
		"input": input,
		"summary": envelope{
			"yearsToRetirement": years, "monthsToRetirement": months,
			"monthlyExpenseAtRetirement": roundMoney(expenseAtRetirement),
			"annualExpenseAtRetirement":  roundMoney(annualExpense),
			"targetFund":                 roundMoney(targetFund), "projectedFund": roundMoney(projectedFund),
			"gap": roundMoney(gap), "progress": progress,
			"requiredMonthlyContribution": roundMoney(requiredContribution),
			"status":                      status, "message": message,
		},
		"projection": points,
	}
}

func requiredMonthlyContribution(target, current, monthlyRate float64, months int) float64 {
	if months <= 0 {
		return 0
	}
	if monthlyRate == 0 {
		return math.Max((target-current)/float64(months), 0)
	}
	growth := math.Pow(1+monthlyRate, float64(months))
	futureCurrent := current * growth
	annuityFactor := (growth - 1) / monthlyRate
	return math.Max((target-futureCurrent)/annuityFactor, 0)
}
