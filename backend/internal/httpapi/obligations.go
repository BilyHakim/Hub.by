package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type obligationInput struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	OriginalAmount   int64  `json:"originalAmount"`
	InstallmentCount int    `json:"installmentCount"`
	StartDate        string `json:"startDate"`
	Notes            string `json:"notes"`
}

type obligationPaymentInput struct {
	Amount    int64  `json:"amount"`
	PaidAt    string `json:"paidAt"`
	Notes     string `json:"notes"`
	AccountID int64  `json:"accountId"`
}

func (api *API) listObligations(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT o.id,o.type,o.name,o.platform,o.original_amount,o.installment_count,
		       o.start_date,o.notes,
		       COALESCE(SUM(p.amount),0)::bigint,COUNT(p.id)::int,
		       (ARRAY_AGG(p.id ORDER BY p.paid_at DESC,p.id DESC) FILTER (WHERE p.id IS NOT NULL))[1],
		       MAX(p.paid_at)
		FROM obligations o
		LEFT JOIN obligation_payments p ON p.obligation_id=o.id
		WHERE o.workspace_id=$1
		GROUP BY o.id
		ORDER BY (COALESCE(SUM(p.amount),0) < o.original_amount) DESC,o.start_date DESC,o.id DESC
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load obligations")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, original, paid int64
		var kind, name, platform, notes string
		var installments, paymentCount int
		var start time.Time
		var lastPaymentID sql.NullInt64
		var lastPaidAt sql.NullTime
		if err := rows.Scan(&id, &kind, &name, &platform, &original, &installments, &start, &notes, &paid, &paymentCount, &lastPaymentID, &lastPaidAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read obligations")
			return
		}
		remaining := max(original-paid, 0)
		progress := ratio(min(paid, original), original)
		expected := (original + int64(installments) - 1) / int64(installments)
		item := envelope{
			"id": id, "type": kind, "name": name, "platform": platform,
			"originalAmount": original, "installmentCount": installments,
			"startDate": start.Format("2006-01-02"), "notes": notes,
			"paidAmount": paid, "remainingAmount": remaining, "progress": progress,
			"paymentCount": paymentCount, "expectedInstallment": expected,
			"lastPaymentId": nil,
		}
		if lastPaymentID.Valid {
			item["lastPaymentId"] = lastPaymentID.Int64
		}
		if lastPaidAt.Valid {
			item["lastPaidAt"] = lastPaidAt.Time.Format("2006-01-02")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read obligations")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) createObligation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input obligationInput
	start, ok := decodeObligationInput(r, &input)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid debt or receivable values")
		return
	}
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO obligations(workspace_id,type,name,platform,original_amount,installment_count,start_date,notes)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, workspaceID, input.Type, input.Name, input.Platform, input.OriginalAmount, input.InstallmentCount, start, input.Notes).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create obligation")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": id}})
}

func (api *API) updateObligation(w http.ResponseWriter, r *http.Request) {
	workspaceID, id, ok := api.obligationIdentity(w, r)
	if !ok {
		return
	}
	var input obligationInput
	start, valid := decodeObligationInput(r, &input)
	if !valid {
		writeError(w, http.StatusUnprocessableEntity, "invalid debt or receivable values")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE obligations SET type=$1,name=$2,platform=$3,original_amount=$4,
		       installment_count=$5,start_date=$6,notes=$7,updated_at=now()
		WHERE id=$8 AND workspace_id=$9
		  AND $4 >= (SELECT COALESCE(SUM(amount),0) FROM obligation_payments WHERE obligation_id=$8)
	`, input.Type, input.Name, input.Platform, input.OriginalAmount, input.InstallmentCount, start, input.Notes, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update obligation")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "obligation not found")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id}})
}

func (api *API) deleteObligation(w http.ResponseWriter, r *http.Request) {
	workspaceID, id, ok := api.obligationIdentity(w, r)
	if !ok {
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start obligation removal")
		return
	}
	defer tx.Rollback(r.Context())

	// Remove the cash-flow records created by payments and restore every affected
	// account before the obligation's cascading payment history is removed.
	_, err = tx.Exec(r.Context(), `
		WITH removed AS (
			DELETE FROM transactions t USING obligation_payments p
			WHERE p.obligation_id=$1 AND p.transaction_id=t.id AND t.workspace_id=$2
			RETURNING t.account_id,t.type,t.amount
		), changes AS (
			SELECT account_id,
			       SUM(CASE WHEN type='income' THEN amount ELSE -amount END)::bigint AS balance_change
			FROM removed GROUP BY account_id
		)
		UPDATE accounts a
		SET current_balance=a.current_balance-c.balance_change,updated_at=now()
		FROM changes c WHERE a.id=c.account_id AND a.workspace_id=$2
	`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove obligation transactions")
		return
	}
	result, err := tx.Exec(r.Context(), `DELETE FROM obligations WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete obligation")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "obligation not found")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit obligation removal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) createObligationPayment(w http.ResponseWriter, r *http.Request) {
	workspaceID, id, ok := api.obligationIdentity(w, r)
	if !ok {
		return
	}
	var input obligationPaymentInput
	if decodeJSON(r, &input) != nil || input.Amount <= 0 || input.AccountID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "positive payment amount and payment account are required")
		return
	}
	paidAt, err := time.Parse("2006-01-02", input.PaidAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "paidAt must use YYYY-MM-DD")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start payment")
		return
	}
	defer tx.Rollback(r.Context())

	var obligationType, obligationName string
	var remaining int64
	err = tx.QueryRow(r.Context(), `
		SELECT o.type,o.name,o.original_amount-(
			SELECT COALESCE(SUM(p.amount),0) FROM obligation_payments p WHERE p.obligation_id=o.id
		)::bigint
		FROM obligations o WHERE o.id=$1 AND o.workspace_id=$2 FOR UPDATE
	`, id, workspaceID).Scan(&obligationType, &obligationName, &remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "obligation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load obligation")
		return
	}
	if input.Amount > remaining {
		writeError(w, http.StatusUnprocessableEntity, "payment exceeds the remaining amount")
		return
	}

	transactionType := "expense"
	preferredCategory := "Cicilan"
	description := "Pembayaran " + obligationName
	isDebtPayment := true
	if obligationType == "receivable" {
		transactionType = "income"
		preferredCategory = "Penerimaan piutang"
		description = "Penerimaan " + obligationName
		isDebtPayment = false
	}
	var categoryID int64
	err = tx.QueryRow(r.Context(), `
		SELECT id FROM categories
		WHERE workspace_id=$1 AND type=$2
		ORDER BY CASE WHEN name=$3 THEN 0 ELSE 1 END,id LIMIT 1
	`, workspaceID, transactionType, preferredCategory).Scan(&categoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "no matching transaction category is available")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve payment category")
		return
	}

	var transactionID, balance int64
	err = tx.QueryRow(r.Context(), `
		WITH inserted AS (
			INSERT INTO transactions(workspace_id,type,category_id,account_id,amount,description,occurred_at,is_debt_payment)
			SELECT $1,$2,$3,a.id,$5,$6,$7,$8 FROM accounts a
			WHERE a.id=$4 AND a.workspace_id=$1 AND a.kind IN ('cash','bank','ewallet')
			RETURNING id,account_id,type,amount
		), updated AS (
			UPDATE accounts a
			SET current_balance=a.current_balance+CASE WHEN i.type='income' THEN i.amount ELSE -i.amount END,
			    updated_at=now()
			FROM inserted i WHERE a.id=i.account_id AND a.workspace_id=$1
			RETURNING i.id,a.current_balance
		)
		SELECT id,current_balance FROM updated
	`, workspaceID, transactionType, categoryID, input.AccountID, input.Amount, description, paidAt, isDebtPayment).Scan(&transactionID, &balance)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "payment account must be a cash, bank, or e-wallet account in the active workspace")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create payment transaction")
		return
	}

	var paymentID int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO obligation_payments(obligation_id,amount,paid_at,notes,transaction_id)
		VALUES($1,$2,$3,$4,$5) RETURNING id
	`, id, input.Amount, paidAt, strings.TrimSpace(input.Notes), transactionID).Scan(&paymentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record payment")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit payment")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": paymentID, "transactionId": transactionID, "accountBalance": balance,
	}})
}

func (api *API) deleteObligationPayment(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payment id")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start payment removal")
		return
	}
	defer tx.Rollback(r.Context())

	var transactionID sql.NullInt64
	err = tx.QueryRow(r.Context(), `
		SELECT p.transaction_id FROM obligation_payments p
		JOIN obligations o ON o.id=p.obligation_id
		WHERE p.id=$1 AND o.workspace_id=$2 FOR UPDATE
	`, id, workspaceID).Scan(&transactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load payment")
		return
	}
	if transactionID.Valid {
		var accountID, amount int64
		var transactionType string
		err = tx.QueryRow(r.Context(), `
			DELETE FROM transactions WHERE id=$1 AND workspace_id=$2
			RETURNING account_id,type,amount
		`, transactionID.Int64, workspaceID).Scan(&accountID, &transactionType, &amount)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "failed to remove payment transaction")
			return
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `
				UPDATE accounts SET current_balance=current_balance-CASE WHEN $1='income' THEN $2 ELSE -$2 END,updated_at=now()
				WHERE id=$3 AND workspace_id=$4
			`, transactionType, amount, accountID, workspaceID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to restore payment account")
				return
			}
		}
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM obligation_payments WHERE id=$1`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete payment")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit payment removal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeObligationInput(r *http.Request, input *obligationInput) (time.Time, bool) {
	if decodeJSON(r, input) != nil {
		return time.Time{}, false
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Platform = strings.TrimSpace(input.Platform)
	input.Notes = strings.TrimSpace(input.Notes)
	start, err := time.Parse("2006-01-02", input.StartDate)
	validType := input.Type == "debt" || input.Type == "receivable"
	return start, err == nil && validType && input.Name != "" && input.OriginalAmount > 0 &&
		input.InstallmentCount >= 1 && input.InstallmentCount <= 360
}

func (api *API) obligationIdentity(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return 0, 0, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid obligation id")
		return 0, 0, false
	}
	return workspaceID, id, true
}
