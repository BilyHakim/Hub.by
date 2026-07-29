package httpapi

import (
	"context"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

const localUserID int64 = 1

type profileInput struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Subtitle    string `json:"subtitle"`
}

type workspaceInput struct {
	Name string `json:"name"`
}

func (api *API) getProfile(w http.ResponseWriter, r *http.Request) {
	var id, currentWorkspaceID int64
	var displayName, email, initials, subtitle string
	err := api.db.QueryRow(r.Context(), `
		SELECT id, display_name, email, initials, subtitle, current_workspace_id
		FROM users WHERE id=$1
	`, localUserID).Scan(&id, &displayName, &email, &initials, &subtitle, &currentWorkspaceID)
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
	`, input.DisplayName, input.Email, initials, input.Subtitle, localUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"id": localUserID, "displayName": input.DisplayName, "email": input.Email,
		"initials": initials, "subtitle": input.Subtitle,
	}})
}

func (api *API) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	rows, err := api.db.Query(r.Context(), `
		SELECT w.id,w.name,w.initials,wm.role
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id=w.id
		WHERE wm.user_id=$1
		ORDER BY w.created_at
	`, localUserID)
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
	`, input.Name, initials, localUserID).Scan(&id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')
		`, id, localUserID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			UPDATE users SET current_workspace_id=$1,updated_at=now() WHERE id=$2
		`, id, localUserID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": id, "name": input.Name, "initials": initials, "role": "owner",
	}})
}

func (api *API) selectWorkspace(w http.ResponseWriter, r *http.Request) {
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
	`, input.WorkspaceID, localUserID)
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
