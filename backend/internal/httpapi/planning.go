package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type pyramidItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	IsCompleted bool   `json:"isCompleted"`
	Tracker     string `json:"tracker"`
}

type pyramidLevel struct {
	Priority    int           `json:"priority"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Progress    float64       `json:"progress"`
	Items       []pyramidItem `json:"items"`
}

var pyramidMeta = map[int][2]string{
	1: {"Arus pendapatan", "Pastikan sumber penghasilan stabil sebelum membangun fondasi berikutnya."},
	2: {"Arus kas sehat", "Jaga pemasukan lebih besar dari pengeluaran secara konsisten."},
	3: {"Dana darurat", "Siapkan bantalan untuk kebutuhan tak terduga dan perubahan kondisi kerja."},
	4: {"Proteksi", "Lindungi kesehatan, jiwa, dan aset utama dari risiko besar."},
	5: {"Kewajiban terkendali", "Kurangi utang konsumtif dan jaga cicilan pada tingkat sehat."},
	6: {"Tujuan dan investasi", "Arahkan surplus ke tujuan yang terukur dan investasi terdiversifikasi."},
	7: {"Pensiun dan warisan", "Bangun kemandirian jangka panjang serta rencana penerusannya."},
}

func (api *API) getPyramid(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	if err := api.ensurePyramidItems(r.Context(), workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare financial pyramid")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT id,priority,title,is_completed,tracker_module
		FROM pyramid_items WHERE workspace_id=$1 ORDER BY priority,id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load financial pyramid")
		return
	}
	defer rows.Close()

	levels := make([]pyramidLevel, 7)
	for index := range levels {
		priority := index + 1
		levels[index] = pyramidLevel{
			Priority: priority, Title: pyramidMeta[priority][0],
			Description: pyramidMeta[priority][1], Items: make([]pyramidItem, 0),
		}
	}
	for rows.Next() {
		var item pyramidItem
		var priority int
		if rows.Scan(&item.ID, &priority, &item.Title, &item.IsCompleted, &item.Tracker) == nil {
			levels[priority-1].Items = append(levels[priority-1].Items, item)
		}
	}
	totalCompleted := 0
	totalItems := 0
	for index := range levels {
		completed := 0
		for _, item := range levels[index].Items {
			if item.IsCompleted {
				completed++
				totalCompleted++
			}
		}
		totalItems += len(levels[index].Items)
		if len(levels[index].Items) > 0 {
			levels[index].Progress = float64(completed) / float64(len(levels[index].Items)) * 100
		}
	}
	overall := 0.0
	if totalItems > 0 {
		overall = float64(totalCompleted) / float64(totalItems) * 100
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"levels": levels, "overallProgress": overall,
		"completedItems": totalCompleted, "totalItems": totalItems,
	}})
}

func (api *API) updatePyramidItem(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pyramid item id")
		return
	}
	var input struct {
		IsCompleted bool `json:"isCompleted"`
	}
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE pyramid_items SET is_completed=$1 WHERE id=$2 AND workspace_id=$3
	`, input.IsCompleted, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update pyramid")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "pyramid item not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "isCompleted": input.IsCompleted}})
}

func (api *API) ensurePyramidItems(ctx context.Context, workspaceID int64) error {
	_, err := api.db.Exec(ctx, `
		INSERT INTO pyramid_items(workspace_id,priority,title,is_completed,tracker_module)
		SELECT $1,defaults.priority,defaults.title,FALSE,defaults.tracker
		FROM (VALUES
			(1,'Punya penghasilan tetap dalam 3 bulan terakhir','cashflow'),
			(1,'Punya penghasilan tetap dalam 6 bulan terakhir','cashflow'),
			(1,'Punya penghasilan tetap dalam 12 bulan terakhir','cashflow'),
			(1,'Punya minimal 1 sumber penghasilan tambahan','cashflow'),
			(2,'Pemasukan lebih besar dari pengeluaran dalam 3 bulan terakhir','cashflow'),
			(2,'Pemasukan lebih besar dari pengeluaran dalam 6 bulan terakhir','cashflow'),
			(2,'Pemasukan lebih besar dari pengeluaran dalam 12 bulan terakhir','cashflow'),
			(2,'Rutin mencatat dan mengevaluasi pengeluaran bulanan','cashflow'),
			(3,'Dana darurat 3 bulan pengeluaran tersedia','emergency-fund'),
			(3,'Dana darurat 6 bulan pengeluaran tersedia','emergency-fund'),
			(3,'Dana darurat 12 bulan pengeluaran tersedia','emergency-fund'),
			(4,'BPJS Kesehatan aktif','protection'),
			(4,'Asuransi kesehatan sesuai kebutuhan tersedia','protection'),
			(4,'Asuransi jiwa tersedia jika memiliki tanggungan','protection'),
			(4,'Aset utama sudah memiliki perlindungan','protection'),
			(5,'Tidak memiliki utang konsumtif berbunga tinggi','debt'),
			(5,'Tagihan kartu kredit dibayar penuh setiap bulan','debt'),
			(5,'Rasio kewajiban terhadap pendapatan di bawah 30%','debt'),
			(5,'Punya rencana pelunasan seluruh kewajiban','debt'),
			(6,'Tujuan keuangan sudah dicatat dengan target waktu','goals'),
			(6,'Berinvestasi rutin setiap bulan','investments'),
			(6,'Portofolio investasi sudah terdiversifikasi','investments'),
			(6,'Melakukan evaluasi dan rebalancing berkala','investments'),
			(7,'Target kebutuhan dana pensiun sudah dihitung','retirement'),
			(7,'Menyisihkan dana pensiun secara rutin','retirement'),
			(7,'Proyeksi pensiun mengikuti pendekatan 4%','retirement'),
			(7,'Dokumen waris dan penerima manfaat sudah ditentukan','retirement')
		) AS defaults(priority,title,tracker)
		ON CONFLICT (workspace_id,priority,title) DO NOTHING
	`, workspaceID)
	return err
}

func (api *API) getFinancialCheckup(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	var income, expense, debt, emergency, liquid, investment, assets int64
	err = api.db.QueryRow(r.Context(), `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type='income'),0)::bigint,
			COALESCE(SUM(amount) FILTER (WHERE type='expense'),0)::bigint,
			COALESCE(SUM(amount) FILTER (WHERE type='expense' AND is_debt_payment),0)::bigint
		FROM transactions WHERE workspace_id=$1 AND to_char(occurred_at,'YYYY-MM')=$2
	`, workspaceID, month).Scan(&income, &expense, &debt)
	if err == nil {
		err = api.db.QueryRow(r.Context(), `
			SELECT
				COALESCE(SUM(current_balance) FILTER (WHERE is_emergency_fund),0)::bigint,
				COALESCE(SUM(current_balance) FILTER (WHERE kind IN ('cash','bank','ewallet')),0)::bigint,
				COALESCE(SUM(current_balance) FILTER (WHERE kind='investment'),0)::bigint,
				COALESCE(SUM(current_balance) FILTER (WHERE kind <> 'liability'),0)::bigint
			FROM accounts WHERE workspace_id=$1
		`, workspaceID).Scan(&emergency, &liquid, &investment, &assets)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to calculate financial check-up")
		return
	}
	savingsRate := ratio(income-expense, income)
	debtRatio := ratio(debt, income)
	emergencyTarget := expense * 6
	emergencyRatio := ratio(emergency, emergencyTarget)
	liquidityMonths := 0.0
	if expense > 0 {
		liquidityMonths = float64(liquid) / float64(expense)
	}
	investmentRatio := ratio(investment, assets)
	items := []envelope{
		{"key": "debt", "label": "Rasio kewajiban terhadap pendapatan", "formula": "Pembayaran kewajiban / pendapatan × 100%", "value": debtRatio, "unit": "%", "recommendation": "Maksimal 30%", "status": status(debtRatio <= 30)},
		{"key": "savings", "label": "Rasio tabungan", "formula": "Sisa arus kas / pendapatan × 100%", "value": savingsRate, "unit": "%", "recommendation": "Minimal 20%", "status": status(savingsRate >= 20)},
		{"key": "emergency", "label": "Kesiapan dana darurat", "formula": "Dana darurat / target 6 bulan × 100%", "value": emergencyRatio, "unit": "%", "recommendation": "Minimal 100%", "status": status(emergencyRatio >= 100)},
		{"key": "liquidity", "label": "Rasio likuiditas", "formula": "Aset likuid / pengeluaran bulanan", "value": liquidityMonths, "unit": " bulan", "recommendation": "Minimal 6 bulan", "status": status(liquidityMonths >= 6)},
		{"key": "investment", "label": "Rasio aset investasi", "formula": "Aset investasi / total aset × 100%", "value": investmentRatio, "unit": "%", "recommendation": "Minimal 20%", "status": status(investmentRatio >= 20)},
	}
	healthy := 0
	for _, item := range items {
		if item["status"] == "healthy" {
			healthy++
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"month": month, "items": items, "healthyCount": healthy, "totalCount": len(items),
	}})
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator) * 100
}

func (api *API) getEmergencyFund(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	var monthlyExpense, targetMonths, currentAmount, observedExpense int64
	err = api.db.QueryRow(r.Context(), `
		SELECT monthly_expense,target_months FROM emergency_fund_settings WHERE workspace_id=$1
	`, workspaceID).Scan(&monthlyExpense, &targetMonths)
	if err == nil {
		err = api.db.QueryRow(r.Context(), `
			SELECT COALESCE(SUM(current_balance),0)::bigint FROM accounts
			WHERE workspace_id=$1 AND is_emergency_fund
		`, workspaceID).Scan(&currentAmount)
	}
	if err == nil {
		err = api.db.QueryRow(r.Context(), `
			SELECT COALESCE(SUM(amount),0)::bigint FROM transactions
			WHERE workspace_id=$1 AND type='expense' AND to_char(occurred_at,'YYYY-MM')=$2
		`, workspaceID, month).Scan(&observedExpense)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load emergency fund")
		return
	}
	target := monthlyExpense * targetMonths
	progress := ratio(currentAmount, target)
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"month": month, "monthlyExpense": monthlyExpense, "observedExpense": observedExpense,
		"targetMonths": targetMonths, "targetAmount": target, "currentAmount": currentAmount,
		"remainingAmount": max(target-currentAmount, 0), "progress": progress,
	}})
}

func (api *API) updateEmergencyFund(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input struct {
		MonthlyExpense int64 `json:"monthlyExpense"`
		TargetMonths   int64 `json:"targetMonths"`
	}
	if decodeJSON(r, &input) != nil || input.MonthlyExpense < 0 || input.TargetMonths < 1 || input.TargetMonths > 24 {
		writeError(w, http.StatusUnprocessableEntity, "monthlyExpense must be positive and targetMonths must be between 1 and 24")
		return
	}
	_, err = api.db.Exec(r.Context(), `
		UPDATE emergency_fund_settings
		SET monthly_expense=$1,target_months=$2,updated_at=now()
		WHERE workspace_id=$3
	`, input.MonthlyExpense, input.TargetMonths, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update emergency fund")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"monthlyExpense": input.MonthlyExpense, "targetMonths": input.TargetMonths,
	}})
}
