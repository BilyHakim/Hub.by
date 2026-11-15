package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type aiSettings struct {
	DefaultCredentialID *int64 `json:"defaultCredentialId"`
	FinanceEnabled      bool   `json:"financeEnabled"`
	IncludeBalances     bool   `json:"includeBalances"`
	IncludeGoals        bool   `json:"includeGoals"`
	IncludeObligations  bool   `json:"includeObligations"`
	RetentionDays       int    `json:"retentionDays"`
	DailyLimit          int    `json:"dailyLimit"`
}

func (api *API) aiIdentity(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	user := authenticatedUserID(r.Context())
	expected, parseErr := strconv.ParseInt(r.Header.Get("X-Hubby-Workspace"), 10, 64)
	if parseErr != nil || expected < 1 {
		writeError(w, http.StatusBadRequest, "workspace wajib dipilih")
		return 0, 0, false
	}
	var workspace int64
	err := api.db.QueryRow(r.Context(), `SELECT u.current_workspace_id FROM users u
		JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=u.current_workspace_id WHERE u.id=$1 AND u.current_workspace_id=$2`, user, expected).Scan(&workspace)
	if err != nil {
		writeError(w, http.StatusForbidden, "workspace tidak tersedia untuk pengguna ini")
		return 0, 0, false
	}
	return workspace, user, true
}

func (api *API) aiLoadSettings(ctx context.Context, workspace, user int64) (aiSettings, error) {
	settings := aiSettings{RetentionDays: 30, DailyLimit: 20}
	err := api.db.QueryRow(ctx, `SELECT default_credential_id,finance_enabled,include_balances,include_goals,include_obligations,retention_days,daily_limit
		FROM ai_settings WHERE workspace_id=$1 AND user_id=$2`, workspace, user).Scan(&settings.DefaultCredentialID, &settings.FinanceEnabled, &settings.IncludeBalances, &settings.IncludeGoals, &settings.IncludeObligations, &settings.RetentionDays, &settings.DailyLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return settings, err
}

func (api *API) getAISettings(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	settings, err := api.aiLoadSettings(r.Context(), workspace, user)
	if err != nil {
		writeError(w, 500, "gagal membaca pengaturan AI")
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT id,name,provider,base_url,api_format,model,max_tokens,zdr,enabled FROM ai_credentials WHERE workspace_id=$1 AND user_id=$2 ORDER BY id`, workspace, user)
	if err != nil {
		writeError(w, 500, "gagal membaca credential AI")
		return
	}
	defer rows.Close()
	credentials := make([]aiCredential, 0)
	for rows.Next() {
		var c aiCredential
		if err = rows.Scan(&c.ID, &c.Name, &c.Provider, &c.BaseURL, &c.Format, &c.Model, &c.MaxTokens, &c.ZDR, &c.Enabled); err != nil {
			writeError(w, 500, "gagal membaca credential AI")
			return
		}
		credentials = append(credentials, c)
	}
	if rows.Err() != nil {
		writeError(w, 500, "gagal membaca credential AI")
		return
	}
	var requests, input, output int64
	err = api.db.QueryRow(r.Context(), `SELECT COUNT(*),COALESCE(SUM(input_tokens),0)::bigint,COALESCE(SUM(output_tokens),0)::bigint FROM ai_runs
		WHERE workspace_id=$1 AND user_id=$2 AND created_at>=now()-interval '24 hours'`, workspace, user).Scan(&requests, &input, &output)
	if err != nil {
		writeError(w, 500, "gagal membaca penggunaan AI")
		return
	}
	writeJSON(w, 200, envelope{"data": envelope{"workspaceId": workspace, "encryptionReady": len(api.aiConfig.key) == 32, "settings": settings, "credentials": credentials, "usage": envelope{"requests": requests, "inputTokens": input, "outputTokens": output}}})
}

func (api *API) saveAISettings(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	var settings aiSettings
	if decodeJSON(r, &settings) != nil || settings.RetentionDays < 1 || settings.RetentionDays > 365 || settings.DailyLimit < 1 || settings.DailyLimit > 200 {
		writeError(w, 422, "retensi harus 1–365 hari dan batas permintaan harus 1–200 per 24 jam")
		return
	}
	if settings.DefaultCredentialID != nil {
		if _, err := api.aiLoadCredential(r.Context(), workspace, user, *settings.DefaultCredentialID); err != nil {
			writeError(w, 422, "pilih credential aktif milik Anda di workspace ini")
			return
		}
	}
	_, err := api.db.Exec(r.Context(), `INSERT INTO ai_settings(workspace_id,user_id,default_credential_id,finance_enabled,include_balances,include_goals,include_obligations,retention_days,daily_limit)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(workspace_id,user_id) DO UPDATE SET default_credential_id=$3,finance_enabled=$4,include_balances=$5,include_goals=$6,include_obligations=$7,retention_days=$8,daily_limit=$9`, workspace, user, settings.DefaultCredentialID, settings.FinanceEnabled, settings.IncludeBalances, settings.IncludeGoals, settings.IncludeObligations, settings.RetentionDays, settings.DailyLimit)
	if err != nil {
		writeError(w, 500, "gagal menyimpan pengaturan AI")
		return
	}
	if err = api.aiCleanup(r.Context(), workspace, user, settings.RetentionDays); err != nil {
		writeError(w, 500, "pengaturan tersimpan, tetapi pembersihan riwayat gagal; muat ulang untuk mencoba kembali")
		return
	}
	writeJSON(w, 200, envelope{"data": settings})
}

func (api *API) aiLoadCredential(ctx context.Context, workspace, user, id int64) (aiCredential, error) {
	var c aiCredential
	err := api.db.QueryRow(ctx, `SELECT id,name,provider,base_url,api_format,model,max_tokens,zdr,enabled,secret FROM ai_credentials WHERE id=$1 AND workspace_id=$2 AND user_id=$3 AND enabled`, id, workspace, user).Scan(&c.ID, &c.Name, &c.Provider, &c.BaseURL, &c.Format, &c.Model, &c.MaxTokens, &c.ZDR, &c.Enabled, &c.Secret)
	return c, err
}

func (api *API) saveAICredential(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	if len(api.aiConfig.key) != 32 {
		writeError(w, 503, "pasang AI_ENCRYPTION_KEY di server sebelum menyimpan credential")
		return
	}
	var input struct {
		aiCredential
		APIKey string `json:"apiKey"`
	}
	if decodeJSON(r, &input) != nil {
		writeError(w, 422, "konfigurasi credential tidak valid")
		return
	}
	c := input.aiCredential
	c.Name, c.Model, c.BaseURL = strings.TrimSpace(c.Name), strings.TrimSpace(c.Model), strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if utf8.RuneCountInString(c.Name) < 2 || len(c.Name) > 120 || c.Model == "" || len(c.Model) > 200 || c.MaxTokens < 256 || c.MaxTokens > 8192 || len(input.APIKey) > 8192 || strings.ContainsAny(input.APIKey, "\r\n") {
		writeError(w, 422, "isi nama, model, dan batas token 256–8192 dengan benar")
		return
	}
	u, err := api.aiConfig.validateURL(c.BaseURL)
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	switch c.Provider {
	case "commandcode":
		if u.Host != "api.commandcode.ai" || u.Path != "/provider/v1" || (c.Format != "chat_completions" && c.Format != "messages") {
			writeError(w, 422, "CommandCode memakai https://api.commandcode.ai/provider/v1 dan format chat atau messages")
			return
		}
	case "openai":
		if u.Host != "api.openai.com" || u.Path != "/v1" || c.Format != "chat_completions" {
			writeError(w, 422, "konfigurasi OpenAI tidak valid")
			return
		}
	case "anthropic":
		if u.Host != "api.anthropic.com" || u.Path != "/v1" || c.Format != "messages" {
			writeError(w, 422, "konfigurasi Anthropic tidak valid")
			return
		}
	case "ollama":
		if c.Format != "ollama" {
			writeError(w, 422, "pilih format Ollama")
			return
		}
	case "custom":
		if c.Format != "chat_completions" && c.Format != "messages" {
			writeError(w, 422, "format API kustom tidak valid")
			return
		}
	default:
		writeError(w, 422, "provider tidak didukung")
		return
	}
	var id int64
	if r.Method == "PUT" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id < 1 {
			writeError(w, 400, "ID credential tidak valid")
			return
		}
	}
	secret, err := api.aiConfig.seal(strings.TrimSpace(input.APIKey), aiSecretScope(workspace, user))
	if err != nil {
		writeError(w, 500, "gagal mengenkripsi credential")
		return
	}
	if id > 0 && input.APIKey == "" {
		var oldURL, oldProvider string
		err = api.db.QueryRow(r.Context(), `SELECT secret,base_url,provider FROM ai_credentials WHERE id=$1 AND workspace_id=$2 AND user_id=$3`, id, workspace, user).Scan(&secret, &oldURL, &oldProvider)
		if err != nil {
			writeError(w, 404, "credential tidak ditemukan")
			return
		}
		if oldURL != c.BaseURL || oldProvider != c.Provider {
			if c.Provider != "ollama" {
				writeError(w, 422, "isi API key baru ketika mengganti provider atau endpoint")
				return
			}
			secret, err = api.aiConfig.seal("", aiSecretScope(workspace, user))
			if err != nil {
				writeError(w, 500, "gagal mengenkripsi credential")
				return
			}
		}
	} else if c.Provider != "ollama" && strings.TrimSpace(input.APIKey) == "" {
		writeError(w, 422, "API key wajib diisi")
		return
	}
	if id == 0 {
		err = api.db.QueryRow(r.Context(), `INSERT INTO ai_credentials(workspace_id,user_id,name,provider,base_url,api_format,model,max_tokens,zdr,enabled,secret)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, workspace, user, c.Name, c.Provider, c.BaseURL, c.Format, c.Model, c.MaxTokens, c.ZDR, c.Enabled, secret).Scan(&id)
	} else {
		var updated int64
		err = api.db.QueryRow(r.Context(), `WITH updated AS (
			UPDATE ai_credentials SET name=$1,provider=$2,base_url=$3,api_format=$4,model=$5,max_tokens=$6,zdr=$7,enabled=$8,secret=$9,updated_at=now()
			WHERE id=$10 AND workspace_id=$11 AND user_id=$12 RETURNING id), cleared AS (
			UPDATE ai_settings SET default_credential_id=NULL WHERE workspace_id=$11 AND user_id=$12 AND default_credential_id IN(SELECT id FROM updated) AND NOT $8)
			SELECT id FROM updated`, c.Name, c.Provider, c.BaseURL, c.Format, c.Model, c.MaxTokens, c.ZDR, c.Enabled, secret, id, workspace, user).Scan(&updated)
	}
	if err != nil {
		writeError(w, 500, "gagal menyimpan credential")
		return
	}
	c.ID = id
	writeJSON(w, 200, envelope{"data": c})
}

func (api *API) deleteAICredential(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, 400, "ID tidak valid")
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM ai_credentials WHERE id=$1 AND workspace_id=$2 AND user_id=$3`, id, workspace, user)
	if err != nil {
		writeError(w, 500, "gagal menghapus credential")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "credential tidak ditemukan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) testAICredential(w http.ResponseWriter, r *http.Request) {
	workspace, user, ok := api.aiIdentity(w, r)
	if !ok {
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	c, err := api.aiLoadCredential(r.Context(), workspace, user, id)
	if err != nil {
		writeError(w, 404, "credential aktif tidak ditemukan")
		return
	}
	key, err := api.aiConfig.open(c.Secret, aiSecretScope(workspace, user))
	if err != nil {
		writeError(w, 503, err.Error())
		return
	}
	path := "/models"
	if c.Format == "ollama" {
		path = "/api/tags"
	}
	data, err := api.aiProviderRequest(r.Context(), c, key, "GET", path, nil)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &result) != nil {
		writeError(w, 502, "daftar model dari provider tidak valid")
		return
	}
	models := make([]string, 0)
	for _, m := range result.Data {
		if len(models) < 200 && len(m.ID) < 200 {
			models = append(models, m.ID)
		}
	}
	for _, m := range result.Models {
		if len(models) < 200 && len(m.Name) < 200 {
			models = append(models, m.Name)
		}
	}
	writeJSON(w, 200, envelope{"data": envelope{"models": models, "message": "Koneksi berhasil. Akses generasi model mengikuti izin dan kuota provider."}})
}
