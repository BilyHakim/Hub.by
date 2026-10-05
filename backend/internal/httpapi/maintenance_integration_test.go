package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMaintenanceDatabaseLifecycle(t *testing.T) {
	databaseURL := os.Getenv("MAINTENANCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set MAINTENANCE_TEST_DATABASE_URL to a migrated disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, workspaceID, otherWorkspaceID int64
	err = pool.QueryRow(ctx, `INSERT INTO users(display_name,email,initials) VALUES('Maintenance QA',$1,'QA') RETURNING id`, fmt.Sprintf("maintenance-%d@test.local", time.Now().UnixNano())).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `UPDATE users SET current_workspace_id=NULL WHERE id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM workspaces WHERE owner_user_id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	}()
	for index, destination := range []*int64{&workspaceID, &otherWorkspaceID} {
		if err = pool.QueryRow(ctx, `INSERT INTO workspaces(name,initials,owner_user_id) VALUES($1,'QA',$2) RETURNING id`, fmt.Sprintf("Maintenance QA %d", index), userID).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	switchWorkspace := func(id int64) {
		t.Helper()
		if _, err = pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id=$2`, id, userID); err != nil {
			t.Fatal(err)
		}
	}
	switchWorkspace(workspaceID)
	api := &API{db: pool}
	call := func(handler http.HandlerFunc, method string, id int64, body string, want int) []byte {
		t.Helper()
		request := httptest.NewRequest(method, "/", strings.NewReader(body))
		request = request.WithContext(context.WithValue(ctx, authContextKey{}, userID))
		if id != 0 {
			request.SetPathValue("id", fmt.Sprint(id))
		}
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != want {
			t.Fatalf("%s id=%d: status %d want %d: %s", method, id, response.Code, want, response.Body.String())
		}
		return response.Body.Bytes()
	}
	createID := func(payload []byte) int64 {
		t.Helper()
		var result struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err = json.Unmarshal(payload, &result); err != nil || result.Data.ID == 0 {
			t.Fatalf("invalid created id: %s", payload)
		}
		return result.Data.ID
	}
	overview := func() struct {
		Items     []maintenanceItem `json:"items"`
		Rules     []maintenanceRule `json:"rules"`
		Histories []envelope        `json:"histories"`
	} { t.Helper(); var result struct {
		Data struct {
			Items     []maintenanceItem `json:"items"`
			Rules     []maintenanceRule `json:"rules"`
			Histories []envelope        `json:"histories"`
		} `json:"data"`
	}; if err = json.Unmarshal(call(api.getMaintenance, "GET", 0, "", 200), &result); err != nil {
		t.Fatal(err)
	}; return result.Data }
	if len(overview().Items) != 0 {
		t.Fatal("new workspace is not empty")
	}
	itemID := createID(call(api.saveMaintenanceItem, "POST", 0, `{"name":"QA AC","category":"Custom category","usageUnit":"km","purchaseDate":"2020-01-01","purchasePrice":0}`, 201))
	ruleBody := `{"name":"Clean AC","kind":"maintenance","intervalValue":3,"intervalUnit":"months","startDate":"2020-01-10","reminderDays":14,"active":true}`
	ruleID := createID(call(api.saveMaintenanceRule, "POST", itemID, ruleBody, 201))
	call(api.completeMaintenance, "POST", ruleID, `{"completedAt":"2020-04-10","cost":100000,"vendor":"QA vendor","notes":"QA note"}`, 201)
	call(api.completeMaintenance, "POST", ruleID, `{"completedAt":"2020-01-10","cost":0}`, 201)
	state := overview()
	if state.Rules[0].LastMaintenance != "2020-04-10" || state.Rules[0].NextDueDate != "2020-07-10" || len(state.Histories) != 2 {
		t.Fatalf("out-of-order completion changed schedule: %+v", state)
	}
	call(api.completeMaintenance, "POST", ruleID, `{"completedAt":"2200-01-01"}`, 422)
	call(api.completeMaintenance, "POST", ruleID, `{"completedAt":"2020-04-10","cost":-1}`, 422)
	call(api.saveMaintenanceRule, "PUT", ruleID, strings.Replace(ruleBody, "Clean AC", "Updated rule", 1), 200)
	if overview().Histories[0]["ruleName"] != "Clean AC" {
		t.Fatal("rule editing changed historical label")
	}
	usageRuleID := createID(call(api.saveMaintenanceRule, "POST", itemID, `{"name":"Change oil","kind":"replacement","usageInterval":2000,"startDate":"2020-01-01","reminderUsage":350,"active":true}`, 201))
	call(api.completeMaintenance, "POST", usageRuleID, `{"completedAt":"2020-04-10"}`, 422)
	call(api.completeMaintenance, "POST", usageRuleID, `{"completedAt":"2020-04-10","usageValue":5000}`, 201)
	call(api.completeMaintenance, "POST", usageRuleID, `{"completedAt":"2020-02-10","usageValue":3000}`, 201)
	call(api.completeMaintenance, "POST", usageRuleID, `{"completedAt":"2020-05-10","usageValue":1000}`, 422)
	state = overview()
	if *state.Items[0].CurrentUsage != 5000 || *state.Rules[1].NextDueUsage != 7000 || len(state.Histories) != 4 {
		t.Fatal("usage history, next threshold or atomic validation failed")
	}
	call(api.saveMaintenanceItem, "PUT", itemID, `{"name":"QA AC","usageUnit":"hours","currentUsage":5000}`, 422)
	call(api.saveMaintenanceItem, "PUT", itemID, `{"name":"QA AC","usageUnit":"km","currentUsage":4000}`, 422)
	call(api.saveMaintenanceItem, "PUT", itemID, `{"name":"QA AC edited","category":"New category","usageUnit":"km","currentUsage":6650,"notes":"edited"}`, 200)
	state = overview()
	if state.Items[0].Category != "New category" || state.Rules[1].Status != "due_soon" {
		t.Fatal("item editing did not update usage reminders")
	}
	switchWorkspace(otherWorkspaceID)
	if len(overview().Items) != 0 || len(overview().Rules) != 0 || len(overview().Histories) != 0 {
		t.Fatal("workspace data leaked")
	}
	call(api.saveMaintenanceItem, "PUT", itemID, `{"name":"Cross workspace","usageUnit":"km"}`, 404)
	call(api.saveMaintenanceRule, "POST", itemID, ruleBody, 404)
	call(api.saveMaintenanceRule, "PUT", ruleID, ruleBody, 404)
	call(api.completeMaintenance, "POST", ruleID, `{"completedAt":"2020-04-10"}`, 404)
	call(api.deleteMaintenanceItem, "DELETE", itemID, "", 404)
	switchWorkspace(workspaceID)
	call(api.deleteMaintenanceItem, "DELETE", itemID, "", 204)
	state = overview()
	if len(state.Items) != 0 || len(state.Rules) != 0 || len(state.Histories) != 0 {
		t.Fatal("item deletion did not cascade")
	}
}
