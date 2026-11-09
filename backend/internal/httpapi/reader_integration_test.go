package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReaderDatabaseLifecycle(t *testing.T) {
	databaseURL := os.Getenv("READER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set READER_TEST_DATABASE_URL to a migrated disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var user, workspace, other int64
	err = pool.QueryRow(ctx, `INSERT INTO users(display_name,email,initials) VALUES('Reader QA',$1,'QA') RETURNING id`, fmt.Sprintf("reader-%d@test.local", time.Now().UnixNano())).Scan(&user)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool.Exec(ctx, `UPDATE users SET current_workspace_id=NULL WHERE id=$1`, user)
		pool.Exec(ctx, `DELETE FROM workspaces WHERE owner_user_id=$1`, user)
		pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, user)
	}()
	for _, destination := range []*int64{&workspace, &other} {
		if err = pool.QueryRow(ctx, `INSERT INTO workspaces(name,initials,owner_user_id) VALUES('Reader QA','QA',$1) RETURNING id`, user).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	switchWorkspace := func(id int64) {
		t.Helper()
		if _, err = pool.Exec(ctx, `UPDATE users SET current_workspace_id=$1 WHERE id=$2`, id, user); err != nil {
			t.Fatal(err)
		}
	}
	switchWorkspace(workspace)
	api := &API{db: pool}
	call := func(handler http.HandlerFunc, method string, id int64, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, "/", strings.NewReader(body)).WithContext(context.WithValue(ctx, authContextKey{}, user))
		req.SetPathValue("id", fmt.Sprint(id))
		res := httptest.NewRecorder()
		handler(res, req)
		if res.Code != want {
			t.Fatalf("%s %d: got %d want %d: %s", method, id, res.Code, want, res.Body.String())
		}
		return res.Body.Bytes()
	}
	var physical int64
	if err = pool.QueryRow(ctx, `INSERT INTO book_titles(workspace_id,title,total_pages) VALUES($1,'Physical only',100) RETURNING id`, workspace).Scan(&physical); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(call(api.getReader, "GET", 0, "", 200), []byte(`"books":[]`)) {
		t.Fatal("physical books leaked into reader")
	}
	upload := func(titleID int64, want int) int64 {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		form.WriteField("titleId", fmt.Sprint(titleID))
		form.WriteField("title", "Digital QA")
		form.WriteField("totalPages", "10")
		file, _ := form.CreateFormFile("file", "qa.pdf")
		file.Write([]byte("%PDF-1.7\nQA"))
		form.Close()
		req := httptest.NewRequest("POST", "/", &body).WithContext(context.WithValue(ctx, authContextKey{}, user))
		req.Header.Set("Content-Type", form.FormDataContentType())
		res := httptest.NewRecorder()
		api.uploadReaderBook(res, req)
		if res.Code != want {
			t.Fatalf("upload: %d %s", res.Code, res.Body.String())
		}
		var result struct{ Data struct{ ID int64 } }
		json.Unmarshal(res.Body.Bytes(), &result)
		return result.Data.ID
	}
	id := upload(0, 201)
	upload(id, 409)
	progress := `{"location":"7","currentPage":7,"timerEnabled":true,"sessionId":"qa-session-123456789","seconds":20}`
	call(api.saveReaderProgress, "PUT", id, progress, 200)
	call(api.saveReaderProgress, "PUT", id, progress, 200)
	call(api.saveReaderProgress, "PUT", id, strings.ReplaceAll(strings.ReplaceAll(progress, `"7"`, `"3"`), `"currentPage":7`, `"currentPage":3`), 200)
	var high, position, count, seconds int
	pool.QueryRow(ctx, `SELECT MAX(end_page),COUNT(*) FROM reading_sessions WHERE title_id=$1`, id).Scan(&high, &count)
	pool.QueryRow(ctx, `SELECT current_page FROM reader_books WHERE title_id=$1`, id).Scan(&position)
	pool.QueryRow(ctx, `SELECT SUM(seconds) FROM reader_time WHERE title_id=$1`, id).Scan(&seconds)
	if high != 7 || position != 3 || count != 1 || seconds != 20 {
		t.Fatalf("retry/backtrack corrupted progress: high=%d position=%d sessions=%d time=%d", high, position, count, seconds)
	}
	call(api.saveReaderProgress, "PUT", id, strings.ReplaceAll(progress, `"currentPage":7`, `"currentPage":11`), 422)
	annotation := call(api.createReaderAnnotation, "POST", id, `{"kind":"note","location":"3","page":3,"note":"Keep this"}`, 201)
	var mark struct{ Data struct{ ID int64 } }
	json.Unmarshal(annotation, &mark)
	call(api.readerFile, "GET", id, "", 200)
	switchWorkspace(other)
	call(api.readerFile, "GET", id, "", 404)
	call(api.saveReaderProgress, "PUT", id, progress, 404)
	call(api.deleteReaderAnnotation, "DELETE", mark.Data.ID, "", 404)
	upload(physical, 404)
	if !bytes.Contains(call(api.getReaderAnnotations, "GET", 0, "", 200), []byte(`"data":[]`)) {
		t.Fatal("annotations leaked across workspaces")
	}
	switchWorkspace(workspace)
	call(api.deleteReaderAnnotation, "DELETE", mark.Data.ID, "", 204)
	call(api.deleteReaderBook, "DELETE", id, "", 204)
	var preserved bool
	pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM book_titles WHERE id=$1)`, id).Scan(&preserved)
	if !preserved {
		t.Fatal("removing file deleted Books entry")
	}
	linked := upload(physical, 201)
	if linked != physical {
		t.Fatal("link created a duplicate Books entry")
	}
	call(api.deleteBookTitle, "DELETE", physical, "", 204)
	call(api.readerFile, "GET", physical, "", 404)
}
