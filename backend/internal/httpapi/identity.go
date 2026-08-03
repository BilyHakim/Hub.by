package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type profileInput struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Subtitle    string `json:"subtitle"`
}

type workspaceInput struct {
	Name string `json:"name"`
}

func (api *API) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	var id, currentWorkspaceID int64
	var displayName, email, initials, subtitle string
	err := api.db.QueryRow(r.Context(), `
		SELECT id, display_name, email, initials, subtitle, current_workspace_id
		FROM users WHERE id=$1
	`, userID).Scan(&id, &displayName, &email, &initials, &subtitle, &currentWorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"id": id, "displayName": displayName, "email": email, "initials": initials,
		"subtitle": subtitle, "currentWorkspaceId": currentWorkspaceID,
	}})
}

func (api *API) updateProfile(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	var input profileInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Email = strings.TrimSpace(input.Email)
	input.Subtitle = strings.TrimSpace(input.Subtitle)
	if input.DisplayName == "" || input.Email == "" || !strings.Contains(input.Email, "@") {
		writeError(w, http.StatusUnprocessableEntity, "name and a valid email are required")
		return
	}
	if input.Subtitle == "" {
		input.Subtitle = "Rencana bersama"
	}
	initials := makeInitials(input.DisplayName)
	_, err := api.db.Exec(r.Context(), `
		UPDATE users SET display_name=$1,email=$2,initials=$3,subtitle=$4,updated_at=now()
		WHERE id=$5
	`, input.DisplayName, input.Email, initials, input.Subtitle, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"id": userID, "displayName": input.DisplayName, "email": input.Email,
		"initials": initials, "subtitle": input.Subtitle,
	}})
}

func (api *API) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	rows, err := api.db.Query(r.Context(), `
		SELECT w.id,w.name,w.initials,wm.role
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id
		WHERE wm.user_id=$1
		ORDER BY w.created_at
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load workspaces")
		return
	}
	defer rows.Close()

	items := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var name, initials, role string
		if rows.Scan(&id, &name, &initials, &role) == nil {
			items = append(items, envelope{"id": id, "name": name, "initials": initials, "role": role})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (api *API) createWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	var input workspaceInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if utf8.RuneCountInString(input.Name) < 2 || utf8.RuneCountInString(input.Name) > 80 {
		writeError(w, http.StatusUnprocessableEntity, "workspace name must contain 2 to 80 characters")
		return
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	defer tx.Rollback(context.Background())

	var id int64
	initials := makeInitials(input.Name)
	err = tx.QueryRow(r.Context(), `
		INSERT INTO workspaces(name,initials,owner_user_id) VALUES($1,$2,$3) RETURNING id
	`, input.Name, initials, userID).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')
		`, id, userID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			UPDATE users SET current_workspace_id=$1,updated_at=now() WHERE id=$2
		`, id, userID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO categories(workspace_id,name,type,color,icon,expense_class) VALUES
			($1,'Gaji','income','#49685c','briefcase',NULL),
			($1,'Freelance','income','#7f9d8e','sparkles',NULL),
			($1,'Makanan','expense','#e8a65d','utensils','essential'),
			($1,'Transportasi','expense','#7894a0','car','essential'),
			($1,'Tempat Tinggal','expense','#d77268','home','essential'),
			($1,'Tagihan','expense','#9a8bb7','receipt','obligation'),
			($1,'Belanja','expense','#b4a464','shopping-bag','discretionary'),
			($1,'Hiburan','expense','#638475','party-popper','discretionary'),
			($1,'Cicilan','expense','#af685f','landmark','obligation')
		`, id)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO accounts(workspace_id,name,kind,current_balance,is_emergency_fund)
			VALUES($1,'Rekening Utama','bank',0,FALSE)
		`, id)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO emergency_fund_settings(workspace_id,monthly_expense,target_months)
			VALUES($1,0,6)
		`, id)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO workspace_finance_settings(workspace_id,period_start_day,period_mode)
			VALUES($1,1,'fixed_day')
		`, id)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO retirement_settings(
				workspace_id,current_age,retirement_age,monthly_expense,inflation_rate,
				expected_return,current_fund,monthly_contribution,withdrawal_rate
			) VALUES($1,25,55,5000000,3,6,100000000,4000000,4)
		`, id)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": id, "name": input.Name, "initials": initials, "role": "owner",
	}})
}

func (api *API) currentWorkspaceID(ctx context.Context) (int64, error) {
	userID := authenticatedUserID(ctx)
	var workspaceID int64
	err := api.db.QueryRow(ctx, `
		SELECT current_workspace_id FROM users WHERE id=$1
	`, userID).Scan(&workspaceID)
	return workspaceID, err
}

func (api *API) selectWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	var input struct {
		WorkspaceID int64 `json:"workspaceId"`
	}
	if decodeJSON(r, &input) != nil || input.WorkspaceID <= 0 {
		writeError(w, http.StatusBadRequest, "valid workspaceId is required")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE users SET current_workspace_id=$1,updated_at=now()
		WHERE id=$2 AND EXISTS (
			SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2
		)
	`, input.WorkspaceID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to select workspace")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"currentWorkspaceId": input.WorkspaceID}})
}

func (api *API) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := authenticatedUserID(r.Context())
	workspaceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || workspaceID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	defer tx.Rollback(context.Background())

	var role, workspaceName string
	err = tx.QueryRow(r.Context(), `
		SELECT wm.role,w.name FROM workspace_members wm
		JOIN workspaces w ON w.id=wm.workspace_id
		WHERE wm.workspace_id=$1 AND wm.user_id=$2 FOR UPDATE OF w
	`, workspaceID, userID).Scan(&role, &workspaceName)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load workspace")
		return
	}
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only the workspace owner can delete it")
		return
	}

	var workspaceCount, memberCount int
	if err = tx.QueryRow(r.Context(), `SELECT COUNT(*) FROM workspace_members WHERE user_id=$1`, userID).Scan(&workspaceCount); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate workspace")
		return
	}
	if workspaceCount <= 1 {
		writeError(w, http.StatusConflict, "the last workspace cannot be deleted")
		return
	}
	if err = tx.QueryRow(r.Context(), `SELECT COUNT(*) FROM workspace_members WHERE workspace_id=$1`, workspaceID).Scan(&memberCount); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to validate workspace members")
		return
	}
	if memberCount > 1 {
		writeError(w, http.StatusConflict, "remove other members before deleting this workspace")
		return
	}

	var fallbackID int64
	var fallbackName, fallbackInitials, fallbackRole string
	err = tx.QueryRow(r.Context(), `
		SELECT w.id,w.name,w.initials,wm.role FROM workspace_members wm
		JOIN workspaces w ON w.id=wm.workspace_id
		WHERE wm.user_id=$1 AND w.id<>$2 ORDER BY w.created_at,w.id LIMIT 1
	`, userID, workspaceID).Scan(&fallbackID, &fallbackName, &fallbackInitials, &fallbackRole)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to select replacement workspace")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE users SET current_workspace_id=$1,updated_at=now() WHERE id=$2 AND current_workspace_id=$3
	`, fallbackID, userID, workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to switch active workspace")
		return
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM workspaces WHERE id=$1 AND owner_user_id=$2`, workspaceID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	var currentID int64
	var currentName, currentInitials, currentRole string
	err = tx.QueryRow(r.Context(), `
		SELECT w.id,w.name,w.initials,wm.role FROM users u
		JOIN workspaces w ON w.id=u.current_workspace_id
		JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.user_id=u.id
		WHERE u.id=$1
	`, userID).Scan(&currentID, &currentName, &currentInitials, &currentRole)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"deletedWorkspaceId": workspaceID, "deletedWorkspaceName": workspaceName,
		"currentWorkspace": envelope{"id": currentID, "name": currentName, "initials": currentInitials, "role": currentRole},
	}})
}

func makeInitials(value string) string {
	words := strings.Fields(value)
	var result []rune
	for _, word := range words {
		for _, char := range word {
			if unicode.IsLetter(char) || unicode.IsDigit(char) {
				result = append(result, unicode.ToUpper(char))
				break
			}
		}
		if len(result) == 2 {
			break
		}
	}
	if len(result) == 0 {
		return "HF"
	}
	return string(result)
}
