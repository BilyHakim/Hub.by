package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

type investmentInput struct {
	AssetType        string  `json:"assetType"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	PurchaseValue    int64   `json:"purchaseValue"`
	CurrentValue     int64   `json:"currentValue"`
	TargetAllocation float64 `json:"targetAllocation"`
}

func (api *API) createInvestment(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input investmentInput
	if decodeJSON(r, &input) != nil || !validInvestmentInput(&input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid investment values")
		return
	}
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO investments(
			workspace_id,asset_type,name,platform,purchase_value,current_value,target_allocation
		) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, workspaceID, input.AssetType, input.Name, input.Platform,
		input.PurchaseValue, input.CurrentValue, input.TargetAllocation).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create investment")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": id}})
}

func (api *API) updateInvestment(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid investment id")
		return
	}
	var input investmentInput
	if decodeJSON(r, &input) != nil || !validInvestmentInput(&input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid investment values")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE investments SET
			asset_type=$1,name=$2,platform=$3,purchase_value=$4,
			current_value=$5,target_allocation=$6,updated_at=now()
		WHERE id=$7 AND workspace_id=$8
	`, input.AssetType, input.Name, input.Platform, input.PurchaseValue,
		input.CurrentValue, input.TargetAllocation, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update investment")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "investment not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id}})
}

func (api *API) deleteInvestment(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid investment id")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		DELETE FROM investments WHERE id=$1 AND workspace_id=$2
	`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete investment")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "investment not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validInvestmentInput(input *investmentInput) bool {
	input.AssetType = strings.TrimSpace(input.AssetType)
	input.Name = strings.TrimSpace(input.Name)
	input.Platform = strings.TrimSpace(input.Platform)
	return input.AssetType != "" && input.Name != "" &&
		input.PurchaseValue >= 0 && input.CurrentValue >= 0 &&
		input.TargetAllocation >= 0 && input.TargetAllocation <= 100
}

func (api *API) getRebalancing(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT id,asset_type,name,current_value,target_allocation
		FROM investments WHERE workspace_id=$1 ORDER BY current_value DESC,id
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load rebalancing")
		return
	}
	defer rows.Close()

	type holding struct {
		ID               int64
		AssetType        string
		Name             string
		CurrentValue     int64
		TargetAllocation float64
	}
	holdings := make([]holding, 0)
	var totalValue int64
	totalTarget := 0.0
	for rows.Next() {
		var item holding
		if rows.Scan(&item.ID, &item.AssetType, &item.Name, &item.CurrentValue, &item.TargetAllocation) == nil {
			holdings = append(holdings, item)
			totalValue += item.CurrentValue
			totalTarget += item.TargetAllocation
		}
	}
	items := make([]envelope, 0, len(holdings))
	for _, item := range holdings {
		currentAllocation := ratio(item.CurrentValue, totalValue)
		targetValue := roundMoney(float64(totalValue) * item.TargetAllocation / 100)
		difference := targetValue - item.CurrentValue
		action := "hold"
		if difference > 0 {
			action = "buy"
		} else if difference < 0 {
			action = "sell"
		}
		items = append(items, envelope{
			"id": item.ID, "assetType": item.AssetType, "name": item.Name,
			"currentValue": item.CurrentValue, "currentAllocation": currentAllocation,
			"targetAllocation": item.TargetAllocation, "targetValue": targetValue,
			"difference": difference, "action": action,
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"totalValue": totalValue, "totalTargetAllocation": totalTarget,
		"isBalancedTarget": totalTarget >= 99.99 && totalTarget <= 100.01,
		"items":            items,
	}})
}
