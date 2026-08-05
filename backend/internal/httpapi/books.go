package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const openLibraryBaseURL = "https://openlibrary.org"

type bookTitleInput struct {
	CatalogID   string `json:"catalogId"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CoverURL    string `json:"coverUrl"`
	PublishYear int    `json:"publishYear"`
	TotalPages  int    `json:"totalPages"`
}

type readingSessionInput struct {
	TitleID     int64  `json:"titleId"`
	ReadAt      string `json:"readAt"`
	CurrentPage int    `json:"currentPage"`
	Notes       string `json:"notes"`
}

type openLibrarySearchResponse struct {
	NumFound int `json:"numFound"`
	Docs     []struct {
		Key                 string   `json:"key"`
		Title               string   `json:"title"`
		AuthorNames         []string `json:"author_name"`
		FirstPublishYear    int      `json:"first_publish_year"`
		CoverID             int64    `json:"cover_i"`
		NumberOfPagesMedian int      `json:"number_of_pages_median"`
	} `json:"docs"`
}

type openLibraryDescription string

func (description *openLibraryDescription) UnmarshalJSON(data []byte) error {
	var text string
	if json.Unmarshal(data, &text) == nil {
		*description = openLibraryDescription(text)
		return nil
	}
	var object struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	*description = openLibraryDescription(object.Value)
	return nil
}

type openLibraryWorkResponse struct {
	Title       string                 `json:"title"`
	Description openLibraryDescription `json:"description"`
	Covers      []int64                `json:"covers"`
}

func (api *API) getBooksOverview(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var totalTitles, readingTitles, completedTitles, totalPages, monthPages int64
	err = api.db.QueryRow(r.Context(), `
		SELECT COUNT(*),COUNT(*) FILTER (WHERE status='reading'),COUNT(*) FILTER (WHERE status='completed'),
		       COALESCE((SELECT SUM(pages_read) FROM reading_sessions WHERE workspace_id=$1),0),
		       COALESCE((SELECT SUM(pages_read) FROM reading_sessions WHERE workspace_id=$1 AND read_at >= date_trunc('month',CURRENT_DATE)::date),0)
		FROM book_titles WHERE workspace_id=$1
	`, workspaceID).Scan(&totalTitles, &readingTitles, &completedTitles, &totalPages, &monthPages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load books summary")
		return
	}

	rows, err := api.db.Query(r.Context(), `
		SELECT b.id,b.title,b.author,b.description,b.cover_url,COALESCE(b.publish_year,0),b.total_pages,b.status,b.catalog_id,
		       COALESCE(stats.current_page,0),COALESCE(stats.pages_read,0),COALESCE(TO_CHAR(stats.last_read_at,'YYYY-MM-DD'),'')
		FROM book_titles b
		LEFT JOIN LATERAL (
			SELECT MAX(end_page) current_page,SUM(pages_read) pages_read,MAX(read_at) last_read_at
			FROM reading_sessions WHERE title_id=b.id
		) stats ON TRUE
		WHERE b.workspace_id=$1
		ORDER BY CASE b.status WHEN 'reading' THEN 0 WHEN 'planned' THEN 1 WHEN 'completed' THEN 2 ELSE 3 END,
		         stats.last_read_at DESC NULLS LAST,b.updated_at DESC
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load books library")
		return
	}
	defer rows.Close()
	titles := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var title, author, description, coverURL, status, catalogID, lastReadAt string
		var publishYear, totalPageCount, currentPage, pagesRead int
		if rows.Scan(&id, &title, &author, &description, &coverURL, &publishYear, &totalPageCount, &status, &catalogID, &currentPage, &pagesRead, &lastReadAt) == nil {
			titles = append(titles, bookEnvelope(id, title, author, description, coverURL, status, catalogID, lastReadAt, publishYear, totalPageCount, currentPage, pagesRead))
		}
	}

	sessions, err := api.listReadingSessions(r, workspaceID, 0, 12)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reading history")
		return
	}
	activityRows, err := api.db.Query(r.Context(), `
		SELECT TO_CHAR(day::date,'YYYY-MM-DD'),COALESCE(SUM(s.pages_read),0),COUNT(s.id)
		FROM generate_series(CURRENT_DATE - 6,CURRENT_DATE,INTERVAL '1 day') day
		LEFT JOIN reading_sessions s ON s.read_at=day::date AND s.workspace_id=$1
		GROUP BY day ORDER BY day
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reading activity")
		return
	}
	defer activityRows.Close()
	dailyActivity := make([]envelope, 0, 7)
	for activityRows.Next() {
		var date string
		var pages, count int64
		if activityRows.Scan(&date, &pages, &count) == nil {
			dailyActivity = append(dailyActivity, envelope{"date": date, "pages": pages, "sessions": count})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"summary": envelope{"totalTitles": totalTitles, "readingTitles": readingTitles, "completedTitles": completedTitles, "totalPages": totalPages, "monthPages": monthPages},
		"titles":  titles, "recentSessions": sessions, "dailyActivity": dailyActivity,
	}})
}

func (api *API) getBookTitleDetail(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid book id")
		return
	}
	var title, author, description, coverURL, status, catalogID, lastReadAt string
	var publishYear, totalPages, currentPage, pagesRead int
	err = api.db.QueryRow(r.Context(), `
		SELECT b.title,b.author,b.description,b.cover_url,COALESCE(b.publish_year,0),b.total_pages,b.status,b.catalog_id,
		       COALESCE(stats.current_page,0),COALESCE(stats.pages_read,0),COALESCE(TO_CHAR(stats.last_read_at,'YYYY-MM-DD'),'')
		FROM book_titles b
		LEFT JOIN LATERAL (SELECT MAX(end_page) current_page,SUM(pages_read) pages_read,MAX(read_at) last_read_at FROM reading_sessions WHERE title_id=b.id) stats ON TRUE
		WHERE b.id=$1 AND b.workspace_id=$2
	`, id, workspaceID).Scan(&title, &author, &description, &coverURL, &publishYear, &totalPages, &status, &catalogID, &currentPage, &pagesRead, &lastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "book not found in active workspace")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load book")
		return
	}
	sessions, err := api.listReadingSessions(r, workspaceID, id, 250)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load reading history")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"title":    bookEnvelope(id, title, author, description, coverURL, status, catalogID, lastReadAt, publishYear, totalPages, currentPage, pagesRead),
		"sessions": sessions,
	}})
}

func (api *API) createBookTitle(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input bookTitleInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid book request body")
		return
	}
	if !validBookTitle(&input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid book values")
		return
	}
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO book_titles(workspace_id,catalog_id,title,author,description,cover_url,publish_year,total_pages)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,0),$8) RETURNING id
	`, workspaceID, input.CatalogID, input.Title, input.Author, input.Description, input.CoverURL, input.PublishYear, input.TotalPages).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "book is already in the library")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to add book")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": bookEnvelope(id, input.Title, input.Author, input.Description, input.CoverURL, "planned", input.CatalogID, "", input.PublishYear, input.TotalPages, 0, 0)})
}

func (api *API) updateBookTitleStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	id, parseErr := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input struct {
		Status string `json:"status"`
	}
	if err != nil || parseErr != nil || decodeJSON(r, &input) != nil || !validBookStatus(input.Status) {
		writeError(w, http.StatusUnprocessableEntity, "invalid book status")
		return
	}
	result, err := api.db.Exec(r.Context(), `UPDATE book_titles SET status=$1,updated_at=now() WHERE id=$2 AND workspace_id=$3`, input.Status, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update book status")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "book not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "status": input.Status}})
}

func (api *API) deleteBookTitle(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	id, parseErr := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || parseErr != nil {
		writeError(w, http.StatusBadRequest, "invalid book id")
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM book_titles WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "book not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) createReadingSession(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input readingSessionInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if input.ReadAt == "" {
		input.ReadAt = time.Now().Format("2006-01-02")
	}
	readAt, dateErr := time.Parse("2006-01-02", input.ReadAt)
	if dateErr != nil || input.TitleID <= 0 || input.CurrentPage <= 0 || len(input.Notes) > 500 {
		writeError(w, http.StatusUnprocessableEntity, "invalid reading session")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record reading session")
		return
	}
	defer tx.Rollback(r.Context())
	var title string
	var totalPages, currentPage int
	err = tx.QueryRow(r.Context(), `
		SELECT b.title,b.total_pages,COALESCE((SELECT MAX(end_page) FROM reading_sessions WHERE title_id=b.id),0)
		FROM book_titles b WHERE b.id=$1 AND b.workspace_id=$2 FOR UPDATE
	`, input.TitleID, workspaceID).Scan(&title, &totalPages, &currentPage)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "book not found in active workspace")
		return
	}
	if err != nil || input.CurrentPage <= currentPage || input.CurrentPage > totalPages {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("current page must be between %d and %d", currentPage+1, totalPages))
		return
	}
	pagesRead := input.CurrentPage - currentPage
	var id int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO reading_sessions(workspace_id,title_id,read_at,start_page,end_page,pages_read,notes)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, workspaceID, input.TitleID, readAt, currentPage, input.CurrentPage, pagesRead, input.Notes).Scan(&id)
	if err == nil {
		status := "reading"
		if input.CurrentPage == totalPages {
			status = "completed"
		}
		_, err = tx.Exec(r.Context(), `UPDATE book_titles SET status=$1,updated_at=now() WHERE id=$2`, status, input.TitleID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to record reading session")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{"id": id, "titleId": input.TitleID, "title": title, "readAt": input.ReadAt, "startPage": currentPage, "endPage": input.CurrentPage, "pagesRead": pagesRead, "notes": input.Notes}})
}

func (api *API) deleteReadingSession(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	id, parseErr := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || parseErr != nil {
		writeError(w, http.StatusBadRequest, "invalid reading session id")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete reading session")
		return
	}
	defer tx.Rollback(r.Context())
	var titleID int64
	err = tx.QueryRow(r.Context(), `DELETE FROM reading_sessions WHERE id=$1 AND workspace_id=$2 RETURNING title_id`, id, workspaceID).Scan(&titleID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "reading session not found in active workspace")
		return
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `
			UPDATE book_titles b SET status=CASE
				WHEN b.status='dropped' THEN b.status
				WHEN COALESCE(s.current_page,0)=0 THEN 'planned'
				WHEN s.current_page >= b.total_pages THEN 'completed'
				ELSE 'reading' END,updated_at=now()
			FROM (SELECT MAX(end_page) current_page FROM reading_sessions WHERE title_id=$1) s
			WHERE b.id=$1 AND b.workspace_id=$2
		`, titleID, workspaceID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete reading session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) searchOpenLibrary(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 || len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search query must contain 2 to 100 characters")
		return
	}
	params := url.Values{"q": {query}, "lang": {"id"}, "limit": {"15"}, "fields": {"key,title,author_name,first_publish_year,cover_i,number_of_pages_median"}}
	var response openLibrarySearchResponse
	if !api.fetchOpenLibrary(w, r, "/search.json?"+params.Encode(), &response) {
		return
	}
	items := make([]envelope, 0, len(response.Docs))
	for _, item := range response.Docs {
		catalogID := strings.TrimPrefix(item.Key, "/works/")
		if !validOpenLibraryWorkID(catalogID) {
			continue
		}
		pages := item.NumberOfPagesMedian
		if pages <= 0 {
			pages = 300
		}
		coverURL := ""
		if item.CoverID > 0 {
			coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg?default=false", item.CoverID)
		}
		items = append(items, envelope{"catalogId": catalogID, "title": item.Title, "author": strings.Join(item.AuthorNames, ", "), "publishYear": item.FirstPublishYear, "totalPages": pages, "coverUrl": coverURL, "openLibraryUrl": openLibraryBaseURL + "/works/" + catalogID})
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"items": items, "totalResults": len(items)}})
}

func (api *API) getOpenLibraryWork(w http.ResponseWriter, r *http.Request) {
	workID := strings.TrimSpace(r.PathValue("workID"))
	if !validOpenLibraryWorkID(workID) {
		writeError(w, http.StatusBadRequest, "invalid Open Library work id")
		return
	}
	var response openLibraryWorkResponse
	if !api.fetchOpenLibrary(w, r, "/works/"+url.PathEscape(workID)+".json", &response) {
		return
	}
	coverURL := ""
	if len(response.Covers) > 0 && response.Covers[0] > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg?default=false", response.Covers[0])
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"catalogId": workID, "title": response.Title, "description": string(response.Description),
		"coverUrl": coverURL, "openLibraryUrl": openLibraryBaseURL + "/works/" + workID,
	}})
}

func (api *API) fetchOpenLibrary(w http.ResponseWriter, r *http.Request, path string, target any) bool {
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, openLibraryBaseURL+path, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create Open Library request")
		return false
	}
	request.Header.Set("User-Agent", "HubbyBooks/1.0 (contact: https://bilyhakim.site)")
	upstream, err := api.httpClient.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Open Library is currently unavailable")
		return false
	}
	defer upstream.Body.Close()
	if upstream.StatusCode == http.StatusNotFound {
		writeError(w, http.StatusNotFound, "book was not found on Open Library")
		return false
	}
	if upstream.StatusCode != http.StatusOK || json.NewDecoder(upstream.Body).Decode(target) != nil {
		writeError(w, http.StatusBadGateway, "Open Library returned an invalid response")
		return false
	}
	return true
}

func (api *API) listReadingSessions(r *http.Request, workspaceID, titleID int64, limit int) ([]envelope, error) {
	query := `SELECT s.id,s.title_id,b.title,TO_CHAR(s.read_at,'YYYY-MM-DD'),s.start_page,s.end_page,s.pages_read,s.notes FROM reading_sessions s JOIN book_titles b ON b.id=s.title_id WHERE s.workspace_id=$1`
	args := []any{workspaceID}
	if titleID > 0 {
		query += ` AND s.title_id=$2`
		args = append(args, titleID)
	}
	query += ` ORDER BY s.read_at DESC,s.id DESC LIMIT ` + strconv.Itoa(limit)
	rows, err := api.db.Query(r.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, sessionTitleID int64
		var title, readAt, notes string
		var startPage, endPage, pagesRead int
		if rows.Scan(&id, &sessionTitleID, &title, &readAt, &startPage, &endPage, &pagesRead, &notes) == nil {
			items = append(items, envelope{"id": id, "titleId": sessionTitleID, "title": title, "readAt": readAt, "startPage": startPage, "endPage": endPage, "pagesRead": pagesRead, "notes": notes})
		}
	}
	return items, rows.Err()
}

func validBookTitle(input *bookTitleInput) bool {
	input.CatalogID = strings.TrimSpace(input.CatalogID)
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.Description = strings.TrimSpace(input.Description)
	input.CoverURL = strings.TrimSpace(input.CoverURL)
	return len(input.Title) > 0 && len(input.Title) <= 200 && len(input.Author) <= 200 && len(input.Description) <= 4000 && len(input.CoverURL) <= 1000 && (input.CatalogID == "" || validOpenLibraryWorkID(input.CatalogID)) && (input.PublishYear == 0 || input.PublishYear >= 1000 && input.PublishYear <= 2200) && input.TotalPages > 0 && input.TotalPages <= 100000
}

func validBookStatus(status string) bool {
	return status == "planned" || status == "reading" || status == "completed" || status == "dropped"
}

func validOpenLibraryWorkID(value string) bool {
	if len(value) < 4 || len(value) > 20 || !strings.HasPrefix(value, "OL") || !strings.HasSuffix(value, "W") {
		return false
	}
	_, err := strconv.ParseUint(value[2:len(value)-1], 10, 64)
	return err == nil
}

func bookEnvelope(id int64, title, author, description, coverURL, status, catalogID, lastReadAt string, publishYear, totalPages, currentPage, pagesRead int) envelope {
	return envelope{"id": id, "title": title, "author": author, "description": description, "coverUrl": coverURL, "status": status, "catalogId": catalogID, "lastReadAt": lastReadAt, "publishYear": publishYear, "totalPages": totalPages, "currentPage": currentPage, "pagesRead": pagesRead}
}
