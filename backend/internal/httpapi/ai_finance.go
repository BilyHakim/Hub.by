package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type aiPeriodSummary struct {
	Month            string   `json:"month"`
	Start            string   `json:"start"`
	End              string   `json:"end"`
	Complete         bool     `json:"complete"`
	Income           int64    `json:"income"`
	Expense          int64    `json:"expense"`
	EmergencyExpense int64    `json:"emergencyExpense"`
	Surplus          int64    `json:"surplus"`
	SavingsRate      *float64 `json:"savingsRate"`
	TransactionCount int64    `json:"transactionCount"`
}

type aiCategory struct {
	Name           string `json:"name"`
	Class          string `json:"class"`
	Planned        int64  `json:"planned"`
	Actual         int64  `json:"actual"`
	PreviousActual int64  `json:"previousActual"`
	Difference     int64  `json:"difference"`
	Remaining      int64  `json:"remaining"`
	ScenarioSaving int64  `json:"scenarioSaving"`
}

type aiFinanceContext struct {
	WorkspaceID      int64             `json:"workspaceId"`
	GeneratedAt      time.Time         `json:"generatedAt"`
	Currency         string            `json:"currency"`
	FinanceEnabled   bool              `json:"financeEnabled"`
	Periods          []aiPeriodSummary `json:"periods"`
	Categories       []aiCategory      `json:"categories"`
	Balances         json.RawMessage   `json:"balances,omitempty"`
	Goals            json.RawMessage   `json:"goals,omitempty"`
	Obligations      json.RawMessage   `json:"obligations,omitempty"`
	ReductionPercent int               `json:"reductionPercent"`
	ScenarioSaving   int64             `json:"scenarioSaving"`
	ScenarioExpense  int64             `json:"scenarioExpense"`
	ScenarioSurplus  int64             `json:"scenarioSurplus"`
	Sources          []string          `json:"sources"`
	Notes            []string          `json:"notes"`
}

func aiPercentOf(amount int64, percent int) int64 {
	return amount/100*int64(percent) + amount%100*int64(percent)/100
}

// This is the only transaction used to read Finance for AI. PostgreSQL enforces
// READ ONLY even if a future query accidentally attempts to modify source data.
func (api *API) beginAIFinanceRead(ctx context.Context) (pgx.Tx, error) {
	tx, err := api.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL statement_timeout = '8s'`); err != nil {
		tx.Rollback(context.Background())
		return nil, err
	}
	return tx, nil
}

func (api *API) aiFinanceContext(ctx context.Context, workspace int64, settings aiSettings, month string, reduction int) (aiFinanceContext, error) {
	result := aiFinanceContext{WorkspaceID: workspace, GeneratedAt: time.Now().UTC(), Currency: "IDR", FinanceEnabled: settings.FinanceEnabled,
		Periods: make([]aiPeriodSummary, 0), Categories: make([]aiCategory, 0), Sources: make([]string, 0), Notes: make([]string, 0), ReductionPercent: reduction}
	if reduction < 0 || reduction > 50 {
		return result, errors.New("persentase simulasi harus 0–50")
	}
	label, err := time.Parse("2006-01", month)
	if err != nil || label.Year() < 1900 || label.Year() > 2200 {
		return result, errors.New("periode tidak valid")
	}
	if !settings.FinanceEnabled {
		result.Notes = append(result.Notes, "Akses Finance dinonaktifkan. Tidak ada data keuangan pribadi dalam konteks ini.")
		return result, nil
	}
	// All finance reads share one snapshot and a read-only transaction.
	tx, err := api.beginAIFinanceRead(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(context.Background())
	periodSetting := financePeriodSetting{Mode: periodModeFixedDay, StartDay: 1}
	err = tx.QueryRow(ctx, `SELECT period_mode,period_start_day FROM workspace_finance_settings WHERE workspace_id=$1`, workspace).Scan(&periodSetting.Mode, &periodSetting.StartDay)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	periods := make([]financePeriod, 0, 4)
	for offset := 0; offset < 4; offset++ {
		period, err := buildFinancePeriod(label.AddDate(0, -offset, 0).Format("2006-01"), periodSetting)
		if err != nil {
			return result, err
		}
		periods = append(periods, period)
		item := aiPeriodSummary{Month: period.Label, Start: period.Start.Format("2006-01-02"), End: period.EndInclusive.Format("2006-01-02"), Complete: !time.Now().Before(period.EndExclusive)}
		err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(t.amount) FILTER(WHERE t.type='income'),0)::bigint,
			COALESCE(SUM(t.amount) FILTER(WHERE t.type='expense'),0)::bigint,
			COALESCE(SUM(t.amount) FILTER(WHERE t.type='expense' AND a.is_emergency_fund),0)::bigint,COUNT(*)
			FROM transactions t JOIN accounts a ON a.id=t.account_id AND a.workspace_id=t.workspace_id
			WHERE t.workspace_id=$1 AND t.occurred_at>=$2 AND t.occurred_at<$3`, workspace, period.Start, period.EndExclusive).Scan(&item.Income, &item.Expense, &item.EmergencyExpense, &item.TransactionCount)
		if err != nil {
			return result, err
		}
		item.Surplus = item.Income - item.Expense
		if item.Income > 0 {
			rate := ratio(item.Surplus, item.Income)
			item.SavingsRate = &rate
		}
		result.Periods = append(result.Periods, item)
	}
	rows, err := tx.Query(ctx, `WITH actual AS (
		SELECT category_id,SUM(amount) FILTER(WHERE occurred_at>=$2 AND occurred_at<$3)::bigint AS amount,
		SUM(amount) FILTER(WHERE occurred_at>=$4 AND occurred_at<$5)::bigint AS previous
		FROM transactions WHERE workspace_id=$1 AND type='expense' AND occurred_at>=$4 AND occurred_at<$3 GROUP BY category_id)
		SELECT c.name,COALESCE(c.expense_class,'unclassified'),COALESCE(b.planned_amount,0),COALESCE(a.amount,0),COALESCE(a.previous,0)
		FROM categories c LEFT JOIN actual a ON a.category_id=c.id
		LEFT JOIN monthly_budgets b ON b.category_id=c.id AND b.workspace_id=$1 AND b.month=$6
		WHERE c.workspace_id=$1 AND c.type='expense' ORDER BY c.id LIMIT 501`, workspace, periods[0].Start, periods[0].EndExclusive, periods[1].Start, periods[1].EndExclusive, label)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item aiCategory
		if err = rows.Scan(&item.Name, &item.Class, &item.Planned, &item.Actual, &item.PreviousActual); err != nil {
			rows.Close()
			return result, err
		}
		item.Remaining, item.Difference = item.Planned-item.Actual, item.Actual-item.PreviousActual
		if item.Class == "discretionary" {
			item.ScenarioSaving = aiPercentOf(item.Actual, reduction)
		}
		result.ScenarioSaving += item.ScenarioSaving
		result.Categories = append(result.Categories, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(result.Categories) > 500 {
		return result, errors.New("terlalu banyak kategori untuk satu analisis")
	}
	result.ScenarioExpense = result.Periods[0].Expense - result.ScenarioSaving
	result.ScenarioSurplus = result.Periods[0].Income - result.ScenarioExpense
	result.Sources = append(result.Sources, "Ringkasan arus kas: periode pilihan dan tiga periode sebelumnya", "Budget dan realisasi per kategori: periode pilihan")
	result.Notes = append(result.Notes,
		"Expense mencakup semua pengeluaran, termasuk emergencyExpense. Transfer antar rekening tidak dihitung sebagai pemasukan/pengeluaran.",
		"Complete menunjukkan periode kalender telah berakhir, bukan jaminan pencatatan lengkap. Jangan membandingkan periode parsial dengan periode penuh tanpa menjelaskan perbedaannya.",
		"Simulasi mengurangi persentase pilihan dari pengeluaran tercatat berkategori discretionary, dibulatkan turun per kategori. Ini skenario periode pilihan, bukan proyeksi atau perubahan budget otomatis.",
		"Deskripsi transaksi, nomor rekening, dan catatan pribadi tidak disertakan. Nama kategori dan tujuan tetap dapat memuat informasi pribadi.")
	if settings.IncludeBalances {
		err = tx.QueryRow(ctx, `SELECT jsonb_build_object('asOf',now(),'assets',COALESCE(SUM(current_balance) FILTER(WHERE kind<>'liability'),0),
			'liabilities',ABS(COALESCE(SUM(current_balance) FILTER(WHERE kind='liability'),0)),
			'emergencyFund',COALESCE(SUM(current_balance) FILTER(WHERE is_emergency_fund),0)) FROM accounts WHERE workspace_id=$1`, workspace).Scan(&result.Balances)
		if err != nil {
			return result, err
		}
		result.Sources = append(result.Sources, "Saldo agregat rekening saat pengambilan data")
	}
	if settings.IncludeGoals {
		err = tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('name',name,'target',target_amount,'current',current_amount,
			'monthlyContribution',monthly_contribution,'targetDate',target_date) ORDER BY id),'[]'::jsonb) FROM financial_goals WHERE workspace_id=$1`, workspace).Scan(&result.Goals)
		if err != nil {
			return result, err
		}
		result.Sources = append(result.Sources, "Tujuan keuangan dan kontribusi yang dicatat")
	}
	if settings.IncludeObligations {
		err = tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('type',o.type,'remaining',GREATEST(o.original_amount-COALESCE(p.paid,0),0),
			'expectedInstallment',(o.original_amount+o.installment_count-1)/o.installment_count)),'[]'::jsonb)
			FROM obligations o LEFT JOIN (SELECT obligation_id,SUM(amount) AS paid FROM obligation_payments GROUP BY obligation_id) p ON p.obligation_id=o.id WHERE o.workspace_id=$1`, workspace).Scan(&result.Obligations)
		if err != nil {
			return result, err
		}
		result.Sources = append(result.Sources, "Sisa utang/piutang dan estimasi cicilan; tidak memuat bunga atau jadwal jatuh tempo")
	}
	if len(result.Goals)+len(result.Obligations) > 100000 {
		return result, errors.New("data tujuan atau kewajiban terlalu besar untuk satu analisis")
	}
	err = tx.Commit(ctx)
	return result, err
}

func (api *API) getAIContext(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	settings, err := api.aiLoadSettings(r.Context(), workspace, user)
	if err != nil {
		writeError(w, 500, "gagal membaca pengaturan")
		return
	}
	reduction, err := strconv.Atoi(r.URL.Query().Get("reduction"))
	if err != nil {
		writeError(w, 422, "persentase simulasi tidak valid")
		return
	}
	data, err := api.aiFinanceContext(r.Context(), workspace, settings, r.URL.Query().Get("month"), reduction)
	if err != nil {
		writeError(w, 422, "konteks Finance tidak dapat dibaca; periksa periode, persentase, dan data")
		return
	}
	writeJSON(w, 200, envelope{"data": data})
}
