package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

type transferInput struct {
	SourceAccountID      int64  `json:"sourceAccountId"`
	DestinationAccountID int64  `json:"destinationAccountId"`
	Amount               int64  `json:"amount"`
	Description          string `json:"description"`
	OccurredAt           string `json:"occurredAt"`
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
		SELECT id,kind,amount,description,occurred_at,is_debt_payment,
		       category_id,category_name,account_id,account_name,account_is_emergency_fund,
		       destination_account_id,destination_account_name
		FROM (
			SELECT t.id,t.type::text AS kind,t.amount,t.description,t.occurred_at,t.is_debt_payment,
			       c.id AS category_id,c.name AS category_name,a.id AS account_id,a.name AS account_name,
			       a.is_emergency_fund AS account_is_emergency_fund,
			       0::bigint AS destination_account_id,''::text AS destination_account_name
			FROM transactions t
			JOIN categories c ON c.id=t.category_id
			JOIN accounts a ON a.id=t.account_id
			WHERE t.occurred_at >= $1 AND t.occurred_at < $2 AND t.workspace_id=$3
			UNION ALL
			SELECT tr.id,'transfer',tr.amount,tr.description,tr.occurred_at,false,
			       0::bigint,'Transfer antar rekening',source.id,source.name,source.is_emergency_fund,destination.id,destination.name
			FROM account_transfers tr
			JOIN accounts source ON source.id=tr.source_account_id
			JOIN accounts destination ON destination.id=tr.destination_account_id
			WHERE tr.occurred_at >= $1 AND tr.occurred_at < $2 AND tr.workspace_id=$3
		) records
		ORDER BY occurred_at DESC,id DESC
	`, period.Start, period.EndExclusive, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transactions")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, amount, categoryID, accountID, destinationAccountID int64
		var kind, description, category, account, destinationAccount string
		var occurred time.Time
		var debt, accountIsEmergencyFund bool
		if err := rows.Scan(&id, &kind, &amount, &description, &occurred, &debt, &categoryID, &category, &accountID, &account, &accountIsEmergencyFund, &destinationAccountID, &destinationAccount); err != nil {
			continue
		}
		item := envelope{
			"id": id, "type": kind, "amount": amount, "description": description,
			"occurredAt": occurred.Format("2006-01-02"), "isDebtPayment": debt,
			"category": envelope{"id": categoryID, "name": category},
			"account":  envelope{"id": accountID, "name": account, "isEmergencyFund": accountIsEmergencyFund},
		}
		if kind == "transfer" {
			item["destinationAccount"] = envelope{"id": destinationAccountID, "name": destinationAccount}
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) listRecentTransactions(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	afterID := int64(0)
	if value := r.URL.Query().Get("afterId"); value != "" {
		afterID, err = strconv.ParseInt(value, 10, 64)
		if err != nil || afterID < 0 {
			writeError(w, http.StatusBadRequest, "afterId must be a non-negative integer")
			return
		}
	}

	rows, err := api.db.Query(r.Context(), `
		SELECT t.id, t.type, t.amount, t.description, t.occurred_at, t.created_at,
		       c.name, a.name
		FROM transactions t
		JOIN categories c ON c.id=t.category_id
		JOIN accounts a ON a.id=t.account_id
		WHERE t.workspace_id=$1 AND t.id > $2
		ORDER BY t.id DESC
		LIMIT 50
	`, workspaceID, afterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load recent transactions")
		return
	}
	defer rows.Close()

	items := make([]envelope, 0)
	for rows.Next() {
		var id, amount int64
		var kind, description, category, account string
		var occurred, created time.Time
		if err := rows.Scan(&id, &kind, &amount, &description, &occurred, &created, &category, &account); err != nil {
			continue
		}
		items = append(items, envelope{
			"id": id, "type": kind, "amount": amount, "description": description,
			"occurredAt": occurred.Format("2006-01-02"), "createdAt": created.Format(time.RFC3339),
			"category": category, "account": account,
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

	var id, balance int64
	err = api.db.QueryRow(r.Context(), `
		WITH inserted AS (
			INSERT INTO transactions(workspace_id,type,category_id,account_id,amount,description,occurred_at,is_debt_payment)
			SELECT $8,$1,c.id,a.id,$4,$5,$6,$7
			FROM categories c CROSS JOIN accounts a
			WHERE c.id=$2 AND a.id=$3 AND c.workspace_id=$8 AND a.workspace_id=$8 AND c.type=$1
			RETURNING id,account_id,type,amount
		), updated AS (
			UPDATE accounts a
			SET current_balance=a.current_balance + CASE WHEN i.type='income' THEN i.amount ELSE -i.amount END,
			    updated_at=now()
			FROM inserted i
			WHERE a.id=i.account_id AND a.workspace_id=$8
			RETURNING i.id,a.current_balance
		)
		SELECT id,current_balance FROM updated
	`, input.Type, input.CategoryID, input.AccountID, input.Amount, input.Description, occurredAt, input.IsDebtPayment, workspaceID).Scan(&id, &balance)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnprocessableEntity, "category or account does not belong to the active workspace")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save transaction")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": id, "accountBalance": balance}})
}

func (api *API) updateTransaction(w http.ResponseWriter, r *http.Request) {
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

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction update")
		return
	}
	defer tx.Rollback(r.Context())

	var oldAccountID, oldAmount int64
	var oldType string
	err = tx.QueryRow(r.Context(), `
		SELECT account_id,type::text,amount FROM transactions
		WHERE id=$1 AND workspace_id=$2 FOR UPDATE
	`, id, workspaceID).Scan(&oldAccountID, &oldType, &oldAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transaction")
		return
	}

	var valid bool
	err = tx.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM categories c CROSS JOIN accounts a
			WHERE c.id=$1 AND a.id=$2 AND c.workspace_id=$3 AND a.workspace_id=$3
			  AND c.type=$4::transaction_type
		)
	`, input.CategoryID, input.AccountID, workspaceID, input.Type).Scan(&valid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate transaction")
		return
	}
	if !valid {
		writeError(w, http.StatusUnprocessableEntity, "category or account does not belong to the active workspace")
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE accounts
		SET current_balance=current_balance-CASE
			WHEN $1::text='income' THEN $2::bigint ELSE -$2::bigint
		END,updated_at=now()
		WHERE id=$3 AND workspace_id=$4
	`, oldType, oldAmount, oldAccountID, workspaceID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			UPDATE transactions SET type=$1::transaction_type,category_id=$2,account_id=$3,amount=$4,
				description=$5,occurred_at=$6,is_debt_payment=$7
			WHERE id=$8 AND workspace_id=$9
		`, input.Type, input.CategoryID, input.AccountID, input.Amount, input.Description, occurredAt, input.IsDebtPayment, id, workspaceID)
	}
	var balance int64
	if err == nil {
		err = tx.QueryRow(r.Context(), `
			UPDATE accounts
			SET current_balance=current_balance+CASE
				WHEN $1::text='income' THEN $2::bigint ELSE -$2::bigint
			END,updated_at=now()
			WHERE id=$3 AND workspace_id=$4
			RETURNING current_balance
		`, input.Type, input.Amount, input.AccountID, workspaceID).Scan(&balance)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		api.logger.Error("update transaction", "transaction_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update transaction")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "accountBalance": balance}})
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
	var balance int64
	err = api.db.QueryRow(r.Context(), `
		WITH removed AS (
			DELETE FROM transactions
			WHERE id=$1 AND workspace_id=$2
			RETURNING id,account_id,type,amount
		), updated AS (
			UPDATE accounts a
			SET current_balance=a.current_balance - CASE WHEN r.type='income' THEN r.amount ELSE -r.amount END,
			    updated_at=now()
			FROM removed r
			WHERE a.id=r.account_id AND a.workspace_id=$2
			RETURNING a.current_balance
		)
		SELECT current_balance FROM updated
	`, id, workspaceID).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete transaction")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) createTransfer(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input transferInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	occurredAt, err := validateTransferInput(input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	input.Description = strings.TrimSpace(input.Description)
	if input.Description == "" {
		input.Description = "Transfer antar rekening"
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transfer")
		return
	}
	defer tx.Rollback(r.Context())

	rows, err := tx.Query(r.Context(), `
		SELECT id,current_balance FROM accounts
		WHERE workspace_id=$1 AND id IN ($2,$3)
		ORDER BY id FOR UPDATE
	`, workspaceID, input.SourceAccountID, input.DestinationAccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to lock transfer accounts")
		return
	}
	balances := make(map[int64]int64, 2)
	for rows.Next() {
		var id, balance int64
		if scanErr := rows.Scan(&id, &balance); scanErr != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "failed to read transfer accounts")
			return
		}
		balances[id] = balance
	}
	rows.Close()
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "failed to read transfer accounts")
		return
	}
	if len(balances) != 2 {
		writeError(w, http.StatusUnprocessableEntity, "source and destination accounts must belong to the active workspace")
		return
	}
	if balances[input.SourceAccountID] < input.Amount {
		writeError(w, http.StatusUnprocessableEntity, "source account balance is insufficient")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		UPDATE accounts
		SET current_balance=current_balance - $1,updated_at=now()
		WHERE id=$2 AND workspace_id=$3
	`, input.Amount, input.SourceAccountID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to debit source account")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE accounts
		SET current_balance=current_balance + $1,updated_at=now()
		WHERE id=$2 AND workspace_id=$3
	`, input.Amount, input.DestinationAccountID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to credit destination account")
		return
	}

	var id int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO account_transfers(
			workspace_id,source_account_id,destination_account_id,amount,description,occurred_at
		) VALUES($1,$2,$3,$4,$5,$6) RETURNING id
	`, workspaceID, input.SourceAccountID, input.DestinationAccountID, input.Amount, input.Description, occurredAt).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save transfer")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transfer")
		return
	}

	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id":                        id,
		"sourceAccountBalance":      balances[input.SourceAccountID] - input.Amount,
		"destinationAccountBalance": balances[input.DestinationAccountID] + input.Amount,
	}})
}

func (api *API) deleteTransfer(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid transfer id")
		return
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transfer removal")
		return
	}
	defer tx.Rollback(r.Context())

	var sourceAccountID, destinationAccountID, amount int64
	err = tx.QueryRow(r.Context(), `
		SELECT source_account_id,destination_account_id,amount
		FROM account_transfers WHERE id=$1 AND workspace_id=$2 FOR UPDATE
	`, id, workspaceID).Scan(&sourceAccountID, &destinationAccountID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load transfer")
		return
	}

	rows, err := tx.Query(r.Context(), `
		SELECT id FROM accounts
		WHERE workspace_id=$1 AND id IN ($2,$3)
		ORDER BY id FOR UPDATE
	`, workspaceID, sourceAccountID, destinationAccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to lock transfer accounts")
		return
	}
	accountCount := 0
	for rows.Next() {
		var accountID int64
		if scanErr := rows.Scan(&accountID); scanErr != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "failed to read transfer accounts")
			return
		}
		accountCount++
	}
	rows.Close()
	if rows.Err() != nil || accountCount != 2 {
		writeError(w, http.StatusInternalServerError, "failed to read transfer accounts")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		UPDATE accounts SET current_balance=current_balance + $1,updated_at=now()
		WHERE id=$2 AND workspace_id=$3
	`, amount, sourceAccountID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore source account")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE accounts SET current_balance=current_balance - $1,updated_at=now()
		WHERE id=$2 AND workspace_id=$3
	`, amount, destinationAccountID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore destination account")
		return
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM account_transfers WHERE id=$1 AND workspace_id=$2`, id, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete transfer")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transfer removal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateTransferInput(input transferInput) (time.Time, error) {
	if input.SourceAccountID <= 0 || input.DestinationAccountID <= 0 {
		return time.Time{}, errors.New("source and destination accounts are required")
	}
	if input.SourceAccountID == input.DestinationAccountID {
		return time.Time{}, errors.New("source and destination accounts must be different")
	}
	if input.Amount <= 0 {
		return time.Time{}, errors.New("transfer amount must be positive")
	}
	occurredAt, err := time.Parse("2006-01-02", input.OccurredAt)
	if err != nil {
		return time.Time{}, errors.New("occurredAt must use YYYY-MM-DD")
	}
	return occurredAt, nil
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
