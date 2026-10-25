package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestObligationDatabaseLifecycle(t *testing.T) {
	databaseURL := os.Getenv("OBLIGATIONS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set OBLIGATIONS_TEST_DATABASE_URL to a migrated disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID, workspaceID, otherWorkspaceID int64
	if err = pool.QueryRow(ctx, `INSERT INTO users(display_name,email,initials) VALUES('Obligation QA',$1,'QA') RETURNING id`, fmt.Sprintf("obligation-%d@test.local", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `UPDATE users SET current_workspace_id=NULL WHERE id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM workspaces WHERE owner_user_id=$1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	}()
	for i, destination := range []*int64{&workspaceID, &otherWorkspaceID} {
		if err = pool.QueryRow(ctx, `INSERT INTO workspaces(name,initials,owner_user_id) VALUES($1,'QA',$2) RETURNING id`, fmt.Sprintf("Obligation QA %d", i), userID).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id=$2`, workspaceID, userID); err != nil {
		t.Fatal(err)
	}
	api := &API{db: pool, logger: slog.Default()}
	call := func(handler http.HandlerFunc, method string, id int64, body string, want int) []byte {
		t.Helper()
		r := httptest.NewRequest(method, "/?month=2026-10", strings.NewReader(body))
		r = r.WithContext(context.WithValue(ctx, authContextKey{}, userID))
		r.SetPathValue("id", fmt.Sprint(id))
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s id=%d: status=%d want=%d: %s", method, id, w.Code, want, w.Body.String())
		}
		return w.Body.Bytes()
	}
	createID := func(body []byte) int64 {
		t.Helper()
		var result struct {
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil || result.Data.ID == 0 {
			t.Fatalf("invalid result: %s", body)
		}
		return result.Data.ID
	}
	var categoryID int64
	for _, kind := range []string{"expense", "income"} {
		if err = pool.QueryRow(ctx, `INSERT INTO categories(workspace_id,name,type) VALUES($1,$2,$3) RETURNING id`, workspaceID, kind, kind).Scan(&categoryID); err != nil {
			t.Fatal(err)
		}
	}
	var dailyID, emergencyID, foreignID, investmentID int64
	for i, destination := range []*int64{&dailyID, &emergencyID, &foreignID, &investmentID} {
		wid, kind := workspaceID, "bank"
		if i == 2 {
			wid = otherWorkspaceID
		}
		if i == 3 {
			kind = "investment"
		}
		if err = pool.QueryRow(ctx, `INSERT INTO accounts(workspace_id,name,kind,current_balance,is_emergency_fund) VALUES($1,$2,$3,2000000,$4) RETURNING id`, wid, fmt.Sprintf("Wallet %d", i), kind, i == 1).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	assertBalances := func(daily, emergency int64) {
		t.Helper()
		for id, want := range map[int64]int64{dailyID: daily, emergencyID: emergency} {
			var got int64
			if err := pool.QueryRow(ctx, `SELECT current_balance FROM accounts WHERE id=$1`, id).Scan(&got); err != nil || got != want {
				t.Fatalf("account %d balance=%d want=%d err=%v", id, got, want, err)
			}
		}
	}
	assertPayment := func(id int64, paid int64, count int) {
		t.Helper()
		var gotPaid int64
		var gotCount int
		if err := pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::bigint,COUNT(*) FROM obligation_payments WHERE obligation_id=$1`, id).Scan(&gotPaid, &gotCount); err != nil || gotPaid != paid || gotCount != count {
			t.Fatalf("payment paid=%d count=%d want=%d/%d err=%v", gotPaid, gotCount, paid, count, err)
		}
	}
	for _, kind := range []string{"debt", "receivable"} {
		id := createID(call(api.createObligation, "POST", 0, fmt.Sprintf(`{"type":%q,"name":"Split QA","originalAmount":2000000,"installmentCount":2,"startDate":"2026-10-01"}`, kind), 201))
		payload := fmt.Sprintf(`{"amount":1000000,"paidAt":"2026-10-10","sources":[{"accountId":%d,"amount":600000},{"accountId":%d,"amount":400000}]}`, dailyID, emergencyID)
		paymentID := createID(call(api.createObligationPayment, "POST", id, payload, 201))
		sign := int64(-1)
		if kind == "receivable" {
			sign = 1
		}
		assertBalances(2000000+sign*600000, 2000000+sign*400000)
		assertPayment(id, 1000000, 1)
		var transactionID int64
		if err = pool.QueryRow(ctx, `SELECT transaction_id FROM obligation_payment_transactions WHERE payment_id=$1 LIMIT 1`, paymentID).Scan(&transactionID); err != nil {
			t.Fatal(err)
		}
		call(api.deleteTransaction, "DELETE", transactionID, "", 409)
		call(api.updateTransaction, "PUT", transactionID, fmt.Sprintf(`{"type":"income","categoryId":%d,"accountId":%d,"amount":1,"occurredAt":"2026-10-10"}`, categoryID, dailyID), 409)
		if kind == "debt" {
			var summary struct {
				Data dashboardSummary `json:"data"`
			}
			// Provide income so the dashboard computes the obligations ratio.
			incomeID := createID(call(api.createTransaction, "POST", 0, fmt.Sprintf(`{"type":"income","categoryId":%d,"accountId":%d,"amount":2000000,"occurredAt":"2026-10-10"}`, categoryID, dailyID), 201))
			if err = json.Unmarshal(call(api.dashboard, "GET", 0, "", 200), &summary); err != nil {
				t.Fatal(err)
			}
			if summary.Data.EmergencyExpense != 400000 || summary.Data.Expense != 600000 || summary.Data.FinancialCheckup[1].Value != 50 {
				t.Fatalf("incorrect split dashboard: %+v", summary.Data)
			}
			call(api.deleteTransaction, "DELETE", incomeID, "", 204)
		}
		call(api.deleteObligationPayment, "DELETE", paymentID, "", 204)
		assertBalances(2000000, 2000000)
		assertPayment(id, 0, 0)
		// A bad second source must roll back the first source, too.
		for _, badID := range []int64{foreignID, investmentID} {
			bad := fmt.Sprintf(`{"amount":1000000,"paidAt":"2026-10-10","sources":[{"accountId":%d,"amount":600000},{"accountId":%d,"amount":400000}]}`, dailyID, badID)
			call(api.createObligationPayment, "POST", id, bad, 422)
			assertBalances(2000000, 2000000)
			assertPayment(id, 0, 0)
		}
		createID(call(api.createObligationPayment, "POST", id, payload, 201))
		call(api.createObligationPayment, "POST", id, fmt.Sprintf(`{"amount":1000001,"accountId":%d,"paidAt":"2026-10-10"}`, dailyID), 422)
		assertPayment(id, 1000000, 1)
		// Legacy one-wallet payload still works alongside a split payment.
		createID(call(api.createObligationPayment, "POST", id, fmt.Sprintf(`{"amount":100000,"accountId":%d,"paidAt":"2026-10-11"}`, dailyID), 201))
		assertPayment(id, 1100000, 2)
		call(api.deleteObligation, "DELETE", id, "", 204)
		assertBalances(2000000, 2000000)
	}
	var remaining int
	if err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE workspace_id=$1`, workspaceID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("orphan transactions=%d err=%v", remaining, err)
	}
	// Concurrent requests must not both spend the same remaining debt.
	id := createID(call(api.createObligation, "POST", 0, `{"type":"debt","name":"Concurrent QA","originalAmount":2000000,"installmentCount":2,"startDate":"2026-10-01"}`, 201))
	start, statuses := make(chan struct{}), make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			r := httptest.NewRequest("POST", "/", strings.NewReader(fmt.Sprintf(`{"amount":1500000,"accountId":%d,"paidAt":"2026-10-10"}`, dailyID)))
			r = r.WithContext(context.WithValue(ctx, authContextKey{}, userID))
			r.SetPathValue("id", fmt.Sprint(id))
			w := httptest.NewRecorder()
			api.createObligationPayment(w, r)
			statuses <- w.Code
		}()
	}
	close(start)
	first, second := <-statuses, <-statuses
	if !((first == 201 && second == 422) || (first == 422 && second == 201)) {
		t.Fatalf("concurrent payment statuses=%d,%d", first, second)
	}
	assertPayment(id, 1500000, 1)
	assertBalances(500000, 2000000)
	// Workspace deletion must still cascade with linked payment transactions.
	if _, err = pool.Exec(ctx, `DELETE FROM workspaces WHERE id=$1`, workspaceID); err != nil {
		t.Fatalf("workspace deletion with linked payments: %v", err)
	}
}
