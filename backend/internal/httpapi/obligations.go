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
	Amount int64  `json:"amount"`
	PaidAt string `json:"paidAt"`
	Notes  string `json:"notes"`
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
	result, err := api.db.Exec(r.Context(), `DELETE FROM obligations WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete obligation")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "obligation not found")
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
	if decodeJSON(r, &input) != nil || input.Amount <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "positive payment amount is required")
		return
	}
	paidAt, err := time.Parse("2006-01-02", input.PaidAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "paidAt must use YYYY-MM-DD")
		return
	}
	var paymentID int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO obligation_payments(obligation_id,amount,paid_at,notes)
		SELECT o.id,$3,$4,$5
		FROM obligations o
		WHERE o.id=$1 AND o.workspace_id=$2
		  AND $3 <= o.original_amount - (
		      SELECT COALESCE(SUM(p.amount),0) FROM obligation_payments p WHERE p.obligation_id=o.id
		  )
		RETURNING id
	`, id, workspaceID, input.Amount, paidAt, strings.TrimSpace(input.Notes)).Scan(&paymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "payment exceeds the remaining amount or obligation was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record payment")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": paymentID}})
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
	result, err := api.db.Exec(r.Context(), `
		DELETE FROM obligation_payments p USING obligations o
		WHERE p.id=$1 AND p.obligation_id=o.id AND o.workspace_id=$2
	`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete payment")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "payment not found")
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
