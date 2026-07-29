package httpapi

import (
	"math"
	"net/http"
	"time"
)

type mortgageInput struct {
	PropertyPrice      int64   `json:"propertyPrice"`
	DownPaymentPercent float64 `json:"downPaymentPercent"`
	TenorYears         int     `json:"tenorYears"`
	FixedRate          float64 `json:"fixedRate"`
	FixedYears         int     `json:"fixedYears"`
	FloatingRate       float64 `json:"floatingRate"`
	MonthlyIncome      int64   `json:"monthlyIncome"`
	OtherInstallments  int64   `json:"otherInstallments"`
	OtherCosts         int64   `json:"otherCosts"`
	StartDate          string  `json:"startDate"`
}

type mortgageInstallment struct {
	Month            int    `json:"month"`
	Date             string `json:"date"`
	Payment          int64  `json:"payment"`
	PrincipalPayment int64  `json:"principalPayment"`
	InterestPayment  int64  `json:"interestPayment"`
	RemainingBalance int64  `json:"remainingBalance"`
	RateType         string `json:"rateType"`
}

func (api *API) getMortgageSimulation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO mortgage_simulations(workspace_id) VALUES($1)
		ON CONFLICT(workspace_id) DO NOTHING
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare mortgage simulation")
		return
	}
	input, err := api.loadMortgageInput(r, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load mortgage simulation")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": calculateMortgage(input)})
}

func (api *API) updateMortgageSimulation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input mortgageInput
	if decodeJSON(r, &input) != nil || !validMortgageInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid mortgage simulation values")
		return
	}
	startDate, _ := time.Parse("2006-01-02", input.StartDate)
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO mortgage_simulations(
			workspace_id,property_price,down_payment_percent,tenor_years,
			fixed_rate,fixed_years,floating_rate,monthly_income,
			other_installments,other_costs,start_date
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(workspace_id) DO UPDATE SET
			property_price=EXCLUDED.property_price,
			down_payment_percent=EXCLUDED.down_payment_percent,
			tenor_years=EXCLUDED.tenor_years,
			fixed_rate=EXCLUDED.fixed_rate,
			fixed_years=EXCLUDED.fixed_years,
			floating_rate=EXCLUDED.floating_rate,
			monthly_income=EXCLUDED.monthly_income,
			other_installments=EXCLUDED.other_installments,
			other_costs=EXCLUDED.other_costs,
			start_date=EXCLUDED.start_date,
			updated_at=now()
	`, workspaceID, input.PropertyPrice, input.DownPaymentPercent, input.TenorYears,
		input.FixedRate, input.FixedYears, input.FloatingRate, input.MonthlyIncome,
		input.OtherInstallments, input.OtherCosts, startDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save mortgage simulation")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": calculateMortgage(input)})
}

func (api *API) loadMortgageInput(r *http.Request, workspaceID int64) (mortgageInput, error) {
	var input mortgageInput
	var startDate time.Time
	err := api.db.QueryRow(r.Context(), `
		SELECT property_price,down_payment_percent,tenor_years,fixed_rate,
		       fixed_years,floating_rate,monthly_income,other_installments,
		       other_costs,start_date
		FROM mortgage_simulations WHERE workspace_id=$1
	`, workspaceID).Scan(
		&input.PropertyPrice, &input.DownPaymentPercent, &input.TenorYears,
		&input.FixedRate, &input.FixedYears, &input.FloatingRate,
		&input.MonthlyIncome, &input.OtherInstallments, &input.OtherCosts, &startDate,
	)
	input.StartDate = startDate.Format("2006-01-02")
	return input, err
}

func validMortgageInput(input mortgageInput) bool {
	if input.PropertyPrice <= 0 || input.DownPaymentPercent < 0 || input.DownPaymentPercent >= 100 {
		return false
	}
	if input.TenorYears < 1 || input.TenorYears > 30 || input.FixedYears < 0 || input.FixedYears > input.TenorYears {
		return false
	}
	if input.FixedRate < 0 || input.FixedRate > 100 || input.FloatingRate < 0 || input.FloatingRate > 100 {
		return false
	}
	if input.MonthlyIncome < 0 || input.OtherInstallments < 0 || input.OtherCosts < 0 {
		return false
	}
	_, err := time.Parse("2006-01-02", input.StartDate)
	return err == nil
}

func calculateMortgage(input mortgageInput) envelope {
	downPayment := float64(input.PropertyPrice) * input.DownPaymentPercent / 100
	principal := float64(input.PropertyPrice) - downPayment
	totalMonths := input.TenorYears * 12
	fixedMonths := min(input.FixedYears*12, totalMonths)
	startDate, _ := time.Parse("2006-01-02", input.StartDate)
	balance := principal
	schedule := make([]mortgageInstallment, 0, totalMonths)
	totalInterest := 0.0
	minPayment := math.MaxFloat64
	maxPayment := 0.0
	fixedPayment := annuityPayment(balance, input.FixedRate, totalMonths)
	floatingPayment := 0.0

	for month := 1; month <= totalMonths; month++ {
		rate := input.FixedRate
		rateType := "fixed"
		payment := fixedPayment
		if month > fixedMonths {
			rate = input.FloatingRate
			rateType = "floating"
			if month == fixedMonths+1 {
				floatingPayment = annuityPayment(balance, input.FloatingRate, totalMonths-fixedMonths)
			}
			payment = floatingPayment
		}
		interest := balance * rate / 100 / 12
		principalPayment := payment - interest
		if principalPayment > balance || month == totalMonths {
			principalPayment = balance
			payment = principalPayment + interest
		}
		balance = math.Max(balance-principalPayment, 0)
		totalInterest += interest
		minPayment = math.Min(minPayment, payment)
		maxPayment = math.Max(maxPayment, payment)
		schedule = append(schedule, mortgageInstallment{
			Month: month, Date: startDate.AddDate(0, month-1, 0).Format("2006-01-02"),
			Payment: roundMoney(payment), PrincipalPayment: roundMoney(principalPayment),
			InterestPayment: roundMoney(interest), RemainingBalance: roundMoney(balance),
			RateType: rateType,
		})
	}
	if minPayment == math.MaxFloat64 {
		minPayment = 0
	}
	availableIncome := input.MonthlyIncome - input.OtherInstallments
	minRatio := paymentRatio(minPayment, availableIncome)
	maxRatio := paymentRatio(maxPayment, availableIncome)
	status, conclusion := mortgageHealth(maxRatio)
	return envelope{
		"input": input,
		"summary": envelope{
			"downPayment": roundMoney(downPayment), "principal": roundMoney(principal),
			"totalInterest":    roundMoney(totalInterest),
			"totalLoanPayment": roundMoney(principal + totalInterest),
			"upfrontCost":      roundMoney(downPayment) + input.OtherCosts,
			"totalCashOut":     roundMoney(downPayment+principal+totalInterest) + input.OtherCosts,
			"fixedPayment":     roundMoney(fixedPayment), "floatingPayment": roundMoney(floatingPayment),
			"minPayment": roundMoney(minPayment), "maxPayment": roundMoney(maxPayment),
			"minRatio": minRatio, "maxRatio": maxRatio,
			"status": status, "conclusion": conclusion,
		},
		"schedule": schedule,
	}
}

func annuityPayment(principal, annualRate float64, months int) float64 {
	if months <= 0 || principal <= 0 {
		return 0
	}
	monthlyRate := annualRate / 100 / 12
	if monthlyRate == 0 {
		return principal / float64(months)
	}
	factor := math.Pow(1+monthlyRate, float64(months))
	return principal * monthlyRate * factor / (factor - 1)
}

func paymentRatio(payment float64, income int64) float64 {
	if income <= 0 {
		return 0
	}
	return payment / float64(income) * 100
}

func mortgageHealth(ratio float64) (string, string) {
	switch {
	case ratio < 30:
		return "healthy", "Rasio cicilan sehat dan masih memberi ruang untuk kebutuhan, tabungan, serta investasi."
	case ratio <= 40:
		return "safe", "Cicilan masih dalam batas aman, tetapi cashflow perlu dikelola dengan disiplin."
	case ratio <= 50:
		return "burdened", "Cicilan mulai membebani. Pertimbangkan menambah DP atau memperpanjang tenor."
	default:
		return "unhealthy", "Rasio cicilan terlalu tinggi dan berisiko mengganggu kebutuhan keuangan lainnya."
	}
}

func roundMoney(value float64) int64 {
	return int64(math.Round(value))
}
