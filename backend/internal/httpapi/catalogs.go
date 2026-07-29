package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type categoryInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type accountInput struct {
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Balance         int64  `json:"balance"`
	IsEmergencyFund bool   `json:"isEmergencyFund"`
}

func (api *API) createCategory(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input categoryInput
	if decodeJSON(r, &input) != nil || !validCategoryInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "name and type (income or expense) are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO categories(workspace_id,name,type) VALUES($1,$2,$3) RETURNING id
	`, workspaceID, input.Name, input.Type).Scan(&id)
	if err != nil {
		writeMutationError(w, err, "category")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": id, "name": input.Name, "type": input.Type, "color": "#49685c", "icon": "circle",
	}})
}

func (api *API) updateCategory(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var input categoryInput
	if decodeJSON(r, &input) != nil || !validCategoryInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "name and type (income or expense) are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	result, err := api.db.Exec(r.Context(), `
		UPDATE categories SET name=$1,type=$2 WHERE id=$3 AND workspace_id=$4
	`, input.Name, input.Type, id, workspaceID)
	if err != nil {
		writeMutationError(w, err, "category")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "category not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "name": input.Name, "type": input.Type}})
}

func (api *API) deleteCategory(w http.ResponseWriter, r *http.Request) {
	api.deleteCatalogResource(w, r, "categories", "category")
}

func (api *API) createAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input accountInput
	if decodeJSON(r, &input) != nil || !validAccountInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "valid account name and kind are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO accounts(workspace_id,name,kind,current_balance,is_emergency_fund)
		VALUES($1,$2,$3,$4,$5) RETURNING id
	`, workspaceID, input.Name, input.Kind, input.Balance, input.IsEmergencyFund).Scan(&id)
	if err != nil {
		writeMutationError(w, err, "account")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": id, "name": input.Name, "kind": input.Kind, "balance": input.Balance,
		"isEmergencyFund": input.IsEmergencyFund,
	}})
}

func (api *API) updateAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	var input accountInput
	if decodeJSON(r, &input) != nil || !validAccountInput(input) {
		writeError(w, http.StatusUnprocessableEntity, "valid account name and kind are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	result, err := api.db.Exec(r.Context(), `
		UPDATE accounts
		SET name=$1,kind=$2,current_balance=$3,is_emergency_fund=$4,updated_at=now()
		WHERE id=$5 AND workspace_id=$6
	`, input.Name, input.Kind, input.Balance, input.IsEmergencyFund, id, workspaceID)
	if err != nil {
		writeMutationError(w, err, "account")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "account not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"id": id, "name": input.Name, "kind": input.Kind, "balance": input.Balance,
		"isEmergencyFund": input.IsEmergencyFund,
	}})
}

func (api *API) deleteAccount(w http.ResponseWriter, r *http.Request) {
	api.deleteCatalogResource(w, r, "accounts", "account")
}

func (api *API) deleteCatalogResource(w http.ResponseWriter, r *http.Request, table, resource string) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+resource+" id")
		return
	}
	query := "DELETE FROM " + table + " WHERE id=$1 AND workspace_id=$2"
	result, err := api.db.Exec(r.Context(), query, id, workspaceID)
	if err != nil {
		writeMutationError(w, err, resource)
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, resource+" not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validCategoryInput(input categoryInput) bool {
	name := strings.TrimSpace(input.Name)
	return len(name) >= 2 && len(name) <= 80 && (input.Type == "income" || input.Type == "expense")
}

func validAccountInput(input accountInput) bool {
	name := strings.TrimSpace(input.Name)
	if len(name) < 2 || len(name) > 80 {
		return false
	}
	validKinds := map[string]bool{
		"cash": true, "bank": true, "ewallet": true, "investment": true,
		"property": true, "liability": true,
	}
	return validKinds[input.Kind]
}

func writeMutationError(w http.ResponseWriter, err error, resource string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			writeError(w, http.StatusConflict, resource+" with the same name already exists in this workspace")
			return
		case "23503":
			writeError(w, http.StatusConflict, resource+" is already used by a transaction and cannot be deleted")
			return
		}
	}
	writeError(w, http.StatusInternalServerError, "failed to save "+resource)
}
