package httpapi

import (
	"net/http"
	"strconv"
	"time"
)

type transactionInput struct {
	Type          string `json:"type"`
	CategoryID    int64  `json:"categoryId"`
	AccountID     int64  `json:"accountId"`
	Amount        int64  `json:"amount"`
	Description   string `json:"description"`
	OccurredAt    string `json:"occurredAt"`
	IsDebtPayment bool   `json:"isDebtPayment"`
}

func (api *API) listTransactions(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	period, err := api.resolveFinancePeriod(r.Context(), workspaceID, month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid finance period")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT t.id, t.type, t.amount, t.description, t.occurred_at, t.is_debt_payment,
		       c.id, c.name, a.id, a.name
		FROM transactions t
		JOIN categories c ON c.id=t.category_id
		JOIN accounts a ON a.id=t.account_id
		WHERE t.occurred_at >= $1 AND t.occurred_at < $2 AND t.workspace_id=$3
		ORDER BY t.occurred_at DESC, t.id DESC
	`, period.Start, period.EndExclusive, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transactions")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, amount, categoryID, accountID int64
		var kind, description, category, account string
		var occurred time.Time
		var debt bool
		if err := rows.Scan(&id, &kind, &amount, &description, &occurred, &debt, &categoryID, &category, &accountID, &account); err != nil {
			continue
		}
		items = append(items, envelope{
			"id": id, "type": kind, "amount": amount, "description": description,
			"occurredAt": occurred.Format("2006-01-02"), "isDebtPayment": debt,
			"category": envelope{"id": categoryID, "name": category},
			"account":  envelope{"id": accountID, "name": account},
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input transactionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if (input.Type != "income" && input.Type != "expense") || input.Amount <= 0 || input.CategoryID <= 0 || input.AccountID <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "type, category, account, and positive amount are required")
		return
	}
	occurredAt, err := time.Parse("2006-01-02", input.OccurredAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "occurredAt must use YYYY-MM-DD")
		return
	}

	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO transactions(workspace_id,type,category_id,account_id,amount,description,occurred_at,is_debt_payment)
		SELECT $8,$1,c.id,a.id,$4,$5,$6,$7
		FROM categories c CROSS JOIN accounts a
		WHERE c.id=$2 AND a.id=$3 AND c.workspace_id=$8 AND a.workspace_id=$8
		RETURNING id
	`, input.Type, input.CategoryID, input.AccountID, input.Amount, input.Description, occurredAt, input.IsDebtPayment, workspaceID).Scan(&id)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "category or account does not belong to the active workspace")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": id}})
}

func (api *API) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM transactions WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete transaction")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) listAccounts(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT id,name,kind,current_balance,is_emergency_fund FROM accounts WHERE workspace_id=$1 ORDER BY id`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load accounts")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, balance int64
		var name, kind string
		var emergency bool
		if rows.Scan(&id, &name, &kind, &balance, &emergency) == nil {
			items = append(items, envelope{"id": id, "name": name, "kind": kind, "balance": balance, "isEmergencyFund": emergency})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) listCategories(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT id,name,type,color,icon,expense_class FROM categories WHERE workspace_id=$1 ORDER BY type DESC,id`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load categories")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var name, kind, color, icon string
		var expenseClass *string
		if rows.Scan(&id, &name, &kind, &color, &icon, &expenseClass) == nil {
			items = append(items, envelope{"id": id, "name": name, "type": kind, "color": color, "icon": icon, "expenseClass": expenseClass})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) listInvestments(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT id,asset_type,name,platform,purchase_value,current_value,target_allocation
		FROM investments WHERE workspace_id=$1 ORDER BY current_value DESC,id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load investments")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, purchase, current int64
		var targetAllocation float64
		var assetType, name, platform string
		if rows.Scan(&id, &assetType, &name, &platform, &purchase, &current, &targetAllocation) == nil {
			gain := current - purchase
			percentage := 0.0
			if purchase > 0 {
				percentage = float64(gain) / float64(purchase) * 100
			}
			items = append(items, envelope{
				"id": id, "assetType": assetType, "name": name, "platform": platform,
				"purchaseValue": purchase, "currentValue": current,
				"targetAllocation": targetAllocation, "gain": gain, "returnPercentage": percentage,
			})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}
