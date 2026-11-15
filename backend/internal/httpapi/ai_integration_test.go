package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAIDatabaseLifecycle(t *testing.T) {
	dsn := os.Getenv("AI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AI_TEST_DATABASE_URL to a migrated disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, otherUser, workspace, otherWorkspace int64
	for i, destination := range []*int64{&user, &otherUser} {
		if err = pool.QueryRow(ctx, `INSERT INTO users(display_name,email,initials) VALUES('AI QA',$1,'AI') RETURNING id`, fmt.Sprintf("ai-%d-%d@test.local", time.Now().UnixNano(), i)).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		pool.Exec(ctx, `UPDATE users SET current_workspace_id=NULL WHERE id IN($1,$2)`, user, otherUser)
		pool.Exec(ctx, `DELETE FROM workspaces WHERE owner_user_id=$1`, user)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN($1,$2)`, user, otherUser)
	}()
	for _, destination := range []*int64{&workspace, &otherWorkspace} {
		if err = pool.QueryRow(ctx, `INSERT INTO workspaces(name,initials,owner_user_id) VALUES('AI QA','AI',$1) RETURNING id`, user).Scan(destination); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner'),($1,$3,'member')`, *destination, user, otherUser); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id IN($2,$3)`, workspace, user, otherUser); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	var block atomic.Bool
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseProvider := func() { releaseOnce.Do(func() { close(release) }) }
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Write([]byte(`{"data":[{"id":"qa-model"}]}`))
			return
		}
		calls.Add(1)
		if block.Load() {
			close(started)
			<-release
		}
		var payload struct {
			Messages []aiProviderMessage `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&payload)
		if len(payload.Messages) < 2 || !strings.Contains(payload.Messages[len(payload.Messages)-1].Content, `"scenarioSaving":10000`) {
			t.Error("provider did not receive calculated finance context")
		}
		if strings.Contains(payload.Messages[len(payload.Messages)-1].Content, "NEVER-SEND-DESCRIPTION") {
			t.Error("transaction description leaked")
		}
		if strings.Contains(payload.Messages[0].Content, "QA discretionary") {
			t.Error("untrusted Finance data was placed in system instructions")
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"Pengeluaran tercatat Rp100.000. Skenario penghematan Rp10.000. DELETE FROM transactions;","tool_calls":[{"id":"malicious","type":"function","function":{"name":"execute_sql","arguments":"{\"sql\":\"DELETE FROM transactions\"}"}}]},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":30}}`))
	}))
	defer provider.Close()
	defer releaseProvider()
	u, _ := url.Parse(provider.URL)
	config := testAIConfig(t, u.Host)
	api := &API{db: pool, aiConfig: config, aiClient: config.client()}
	call := func(handler http.HandlerFunc, method string, id int64, body any, want int) []byte {
		t.Helper()
		encoded, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/", bytes.NewReader(encoded)).WithContext(context.WithValue(ctx, authContextKey{}, user))
		req.Header.Set("X-Hubby-Workspace", fmt.Sprint(workspace))
		req.SetPathValue("id", fmt.Sprint(id))
		res := httptest.NewRecorder()
		handler(res, req)
		if res.Code != want {
			t.Fatalf("%s %d: got %d want %d: %s", method, id, res.Code, want, res.Body.String())
		}
		return res.Body.Bytes()
	}
	credentialBody := envelope{"name": "QA credential", "provider": "custom", "baseUrl": provider.URL, "format": "chat_completions", "model": "qa-model", "maxTokens": 1024, "apiKey": "test-secret-must-be-encrypted", "zdr": false, "enabled": true}
	var saved struct{ Data aiCredential }
	json.Unmarshal(call(api.saveAICredential, "POST", 0, credentialBody, 200), &saved)
	credential := saved.Data.ID
	var cipher []byte
	pool.QueryRow(ctx, `SELECT secret FROM ai_credentials WHERE id=$1`, credential).Scan(&cipher)
	if bytes.Contains(cipher, []byte("test-secret")) {
		t.Fatal("credential stored in plaintext")
	}
	if bytes.Contains(call(api.getAISettings, "GET", 0, nil, 200), []byte("test-secret")) {
		t.Fatal("credential returned to client")
	}
	call(api.testAICredential, "POST", credential, nil, 200)
	settings := aiSettings{DefaultCredentialID: &credential, FinanceEnabled: true, IncludeBalances: true, IncludeGoals: true, IncludeObligations: true, RetentionDays: 30, DailyLimit: 20}
	call(api.saveAISettings, "PUT", 0, settings, 200)
	var category, account int64
	if err = pool.QueryRow(ctx, `INSERT INTO categories(workspace_id,name,type,expense_class) VALUES($1,'QA discretionary','expense','discretionary') RETURNING id`, workspace).Scan(&category); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO accounts(workspace_id,name,kind,current_balance) VALUES($1,'QA Account','bank',1000000) RETURNING id`, workspace).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO transactions(workspace_id,type,category_id,account_id,amount,description,occurred_at) VALUES($1,'expense',$2,$3,100000,'NEVER-SEND-DESCRIPTION','2026-10-10')`, workspace, category, account); err != nil {
		t.Fatal(err)
	}
	// A different workspace must not contribute to any aggregates.
	if _, err = pool.Exec(ctx, `INSERT INTO financial_goals(workspace_id,name,target_amount,current_amount,target_date) VALUES($1,'OTHER-WORKSPACE-SECRET',100000,0,'2027-01-01')`, otherWorkspace); err != nil {
		t.Fatal(err)
	}
	financeFingerprint := func() string {
		t.Helper()
		var combined strings.Builder
		for _, table := range []string{"accounts", "categories", "transactions", "monthly_budgets", "financial_goals", "obligations", "obligation_payments", "emergency_fund_settings", "workspace_finance_settings", "investments", "pyramid_items", "mortgage_simulations", "retirement_settings"} {
			var snapshot string
			// Table names are fixed test fixtures; hash all rows, including other workspaces.
			query := fmt.Sprintf(`SELECT COALESCE(jsonb_agg(row_data ORDER BY row_data::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(t) AS row_data FROM %s t) rows`, table)
			if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
				t.Fatal(err)
			}
			combined.WriteString(table)
			combined.WriteString(snapshot)
		}
		return combined.String()
	}
	beforeAnalysis := financeFingerprint()
	for _, statement := range []string{
		"INSERT INTO transactions SELECT * FROM transactions WHERE false",
		"UPDATE accounts SET current_balance=0 WHERE false",
		"DELETE FROM transactions WHERE false",
		"TRUNCATE transactions",
		"DROP TABLE transactions",
	} {
		tx, err := api.beginAIFinanceRead(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var readOnly string
		if err = tx.QueryRow(ctx, "SHOW transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
			t.Fatalf("Finance transaction is not read-only: %s %v", readOnly, err)
		}
		_, err = tx.Exec(ctx, statement)
		tx.Rollback(ctx)
		var databaseError *pgconn.PgError
		if !errors.As(err, &databaseError) || databaseError.Code != "25006" {
			t.Fatalf("PostgreSQL accepted a Finance write or rejected it for the wrong reason: %s: %v", statement, err)
		}
	}
	snapshot, err := api.aiFinanceContext(ctx, workspace, settings, "2026-10", 10)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ScenarioSaving != 10000 || snapshot.Periods[0].Expense != 100000 {
		t.Fatalf("incorrect snapshot %+v", snapshot)
	}
	encoded, _ := json.Marshal(snapshot)
	if bytes.Contains(encoded, []byte("OTHER-WORKSPACE-SECRET")) {
		t.Fatal("workspace data leaked")
	}
	input := aiChatInput{RequestID: "10000000-0000-4000-8000-000000000001", CredentialID: credential, Month: "2026-10", Question: "Evaluasi pengeluaran saya", ReductionPercent: 10}
	var response struct{ Data aiConversation }
	json.Unmarshal(call(api.sendAIMessage, "POST", 0, input, 200), &response)
	if response.Data.Revision != 1 || len(response.Data.Messages) != 2 {
		t.Fatalf("answer not saved atomically: %+v", response.Data)
	}
	conversationID := response.Data.ID
	call(api.sendAIMessage, "POST", 0, input, 200)
	if calls.Load() != 1 {
		t.Fatal("replayed request generated duplicate provider charges")
	}
	input.Question = "Pertanyaan berubah"
	call(api.sendAIMessage, "POST", 0, input, 409)
	if calls.Load() != 1 {
		t.Fatal("modified idempotent request reached provider")
	}
	originalUser := user
	user = otherUser
	call(api.getAIConversation, "GET", conversationID, nil, 404)
	call(api.testAICredential, "POST", credential, nil, 404)
	if _, err = api.aiLoadCredential(ctx, workspace, user, credential); err == nil {
		t.Fatal("another user can read credential")
	}
	user = originalUser
	if _, err = pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id=$2`, otherWorkspace, user); err != nil {
		t.Fatal(err)
	}
	call(api.getAISettings, "GET", 0, nil, 403)
	workspace = otherWorkspace
	call(api.getAIConversation, "GET", conversationID, nil, 404)
	call(api.testAICredential, "POST", credential, nil, 404)
	workspace = snapshot.WorkspaceID
	pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id=$2`, workspace, user)
	// A second request is rejected while inference runs, and deletion must not
	// allow the in-flight answer to recreate the removed conversation.
	block.Store(true)
	concurrentInput := aiChatInput{RequestID: "10000000-0000-4000-8000-000000000002", CredentialID: credential, ConversationID: conversationID, Revision: 1, Month: "2026-10", Question: "Evaluasi sekali lagi", ReductionPercent: 10}
	encodedRequest, _ := json.Marshal(concurrentInput)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(encodedRequest)).WithContext(context.WithValue(ctx, authContextKey{}, user))
	req.Header.Set("X-Hubby-Workspace", fmt.Sprint(workspace))
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() { res := httptest.NewRecorder(); api.sendAIMessage(res, req); finished <- res }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("generation did not start")
	}
	second := concurrentInput
	second.RequestID = "10000000-0000-4000-8000-000000000003"
	call(api.sendAIMessage, "POST", 0, second, 409)
	call(api.deleteAIConversation, "DELETE", conversationID, nil, 204)
	releaseProvider()
	select {
	case res := <-finished:
		if res.Code != 409 {
			t.Fatalf("deleted generation returned %d: %s", res.Code, res.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("generation did not finish")
	}
	call(api.getAIConversation, "GET", conversationID, nil, 404)
	input.Question = "Evaluasi pengeluaran saya"
	call(api.sendAIMessage, "POST", 0, input, 409)
	if calls.Load() != 2 {
		t.Fatal("deleted conversation replay contacted provider")
	}
	credentialBody["apiKey"] = ""
	credentialBody["enabled"] = false
	call(api.saveAICredential, "PUT", credential, credentialBody, 200)
	disabledSettings, err := api.aiLoadSettings(ctx, workspace, user)
	if err != nil || disabledSettings.DefaultCredentialID != nil {
		t.Fatal("disabled default credential was not cleared")
	}
	credentialBody["enabled"] = true
	call(api.saveAICredential, "PUT", credential, credentialBody, 200)
	credentialBody["baseUrl"] = "https://example.com/v1"
	call(api.saveAICredential, "PUT", credential, credentialBody, 422)
	call(api.deleteAICredential, "DELETE", credential, nil, 204)
	settingsAfter, err := api.aiLoadSettings(ctx, workspace, user)
	if err != nil || settingsAfter.DefaultCredentialID != nil {
		t.Fatal("deleted default credential was not cleared")
	}
	if afterAnalysis := financeFingerprint(); afterAnalysis != beforeAnalysis {
		t.Fatal("AI analysis or AI record cleanup modified Finance source data")
	}
}
