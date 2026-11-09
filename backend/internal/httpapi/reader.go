package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const maxReaderFileBytes = 50 << 20

type readerProgress struct {
	Location     string `json:"location"`
	CurrentPage  int    `json:"currentPage"`
	TimerEnabled bool   `json:"timerEnabled"`
	SessionID    string `json:"sessionId"`
	Seconds      int    `json:"seconds"`
}
type readerRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type readerAnnotation struct {
	Kind     string       `json:"kind"`
	Location string       `json:"location"`
	Page     int          `json:"page"`
	Quote    string       `json:"quote"`
	Note     string       `json:"note"`
	Rects    []readerRect `json:"rects"`
}

func validReaderFile(name string, data []byte) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		if bytes.HasPrefix(data, []byte("%PDF-")) {
			return "pdf"
		}
	case ".epub":
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil || len(archive.File) > 10000 {
			return ""
		}
		var size uint64
		validMime, container := false, false
		for _, f := range archive.File {
			size += f.UncompressedSize64
			if f.UncompressedSize64 > 100<<20 || size > 200<<20 {
				return ""
			}
			if f.Name == "META-INF/container.xml" {
				container = true
			}
			if f.Name == "mimetype" {
				source, err := f.Open()
				if err != nil {
					return ""
				}
				mime, err := io.ReadAll(io.LimitReader(source, 64))
				source.Close()
				validMime = err == nil && string(mime) == "application/epub+zip"
			}
		}
		if validMime && container {
			return "epub"
		}
	}
	return ""
}

func (api *API) getReader(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT b.id,b.title,b.author,b.total_pages,d.format,d.filename,d.location,d.current_page,d.timer_enabled,
 COALESCE(TO_CHAR(d.last_read_at,'YYYY-MM-DD HH24:MI'),''),COALESCE((SELECT SUM(seconds) FROM reader_time WHERE title_id=b.id),0)
 FROM reader_books d JOIN book_titles b ON b.id=d.title_id WHERE d.workspace_id=$1 ORDER BY d.last_read_at DESC NULLS LAST,b.updated_at DESC`, workspace)
	if err != nil {
		writeError(w, 500, "failed to load digital library")
		return
	}
	defer rows.Close()
	books := []envelope{}
	for rows.Next() {
		var id, seconds int64
		var title, author, format, filename, location, lastRead string
		var total, page int
		var timer bool
		if err = rows.Scan(&id, &title, &author, &total, &format, &filename, &location, &page, &timer, &lastRead, &seconds); err != nil {
			writeError(w, 500, "failed to load digital library")
			return
		}
		books = append(books, envelope{"id": id, "title": title, "author": author, "totalPages": total, "format": format, "filename": filename, "location": location, "currentPage": page, "timerEnabled": timer, "lastReadAt": lastRead, "seconds": seconds})
	}
	if rows.Err() != nil {
		writeError(w, 500, "failed to load digital library")
		return
	}
	writeJSON(w, 200, envelope{"data": envelope{"books": books}})
}

func (api *API) uploadReaderBook(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	if err = r.ParseMultipartForm(maxReaderFileBytes + (1 << 20)); err != nil {
		writeError(w, 413, "Upload maksimal 50 MB.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 422, "Pilih file PDF atau EPUB.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxReaderFileBytes+1))
	if err != nil || len(data) > maxReaderFileBytes {
		writeError(w, 413, "Upload maksimal 50 MB.")
		return
	}
	format := validReaderFile(header.Filename, data)
	if format == "" {
		writeError(w, 422, "File PDF/EPUB tidak valid atau terlalu besar setelah diekstrak.")
		return
	}
	titleID, _ := strconv.ParseInt(r.FormValue("titleId"), 10, 64)
	title := strings.TrimSpace(r.FormValue("title"))
	author := strings.TrimSpace(r.FormValue("author"))
	total, _ := strconv.Atoi(r.FormValue("totalPages"))
	if titleID < 0 || (titleID == 0 && (title == "" || len(title) > 200 || len(author) > 200 || total < 1 || total > 100000)) {
		writeError(w, 422, "Judul dan jumlah halaman tidak valid.")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "failed to upload book")
		return
	}
	defer tx.Rollback(r.Context())
	if titleID == 0 {
		err = tx.QueryRow(r.Context(), `INSERT INTO book_titles(workspace_id,title,author,total_pages) VALUES($1,$2,$3,$4) RETURNING id`, workspace, title, author, total).Scan(&titleID)
	} else {
		err = tx.QueryRow(r.Context(), `SELECT id FROM book_titles WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, titleID, workspace).Scan(&titleID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "book not found in active workspace")
		return
	}
	if err != nil {
		writeError(w, 500, "failed to upload book")
		return
	}
	var exists bool
	err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM reader_books WHERE title_id=$1)`, titleID).Scan(&exists)
	if err != nil {
		writeError(w, 500, "failed to upload book")
		return
	}
	if exists {
		writeError(w, 409, "Buku ini sudah memiliki file digital.")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO reader_books(title_id,workspace_id,filename,format,content) VALUES($1,$2,$3,$4,$5)`, titleID, workspace, header.Filename, format, data)
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, 500, "failed to upload book")
		return
	}
	writeJSON(w, 201, envelope{"data": envelope{"id": titleID}})
}

func (api *API) readerFile(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var data []byte
	var format string
	err = api.db.QueryRow(r.Context(), `SELECT content,format FROM reader_books WHERE title_id=$1 AND workspace_id=$2`, id, workspace).Scan(&data, &format)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "digital book not found")
		return
	}
	if err != nil {
		writeError(w, 500, "failed to load file")
		return
	}
	w.Header().Set("Content-Type", map[string]string{"pdf": "application/pdf", "epub": "application/epub+zip"}[format])
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `attachment; filename="book.`+format+`"`)
	w.Write(data)
}

func validReaderProgress(input readerProgress) bool {
	return input.CurrentPage > 0 && input.CurrentPage <= 100000 && len(input.Location) > 0 && len(input.Location) <= 2000 && len(input.SessionID) >= 16 && len(input.SessionID) <= 80 && input.Seconds >= 0 && input.Seconds <= 86400
}

func (api *API) saveReaderProgress(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input readerProgress
	if decodeJSON(r, &input) != nil || !validReaderProgress(input) {
		writeError(w, 422, "invalid reading progress")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "failed to save progress")
		return
	}
	defer tx.Rollback(r.Context())
	var total, previous int
	err = tx.QueryRow(r.Context(), `SELECT b.total_pages,COALESCE((SELECT MAX(end_page) FROM reading_sessions WHERE title_id=b.id),0)
 FROM book_titles b JOIN reader_books d ON d.title_id=b.id WHERE b.id=$1 AND d.workspace_id=$2 FOR UPDATE OF b`, id, workspace).Scan(&total, &previous)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "digital book not found")
		return
	}
	if err != nil {
		writeError(w, 500, "failed to save progress")
		return
	}
	if input.CurrentPage > total {
		writeError(w, 422, "page exceeds total pages")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE reader_books SET location=$1,current_page=$2,timer_enabled=$3,last_read_at=now() WHERE title_id=$4`, input.Location, input.CurrentPage, input.TimerEnabled, id)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO reader_time(title_id,session_id,seconds) VALUES($1,$2,$3) ON CONFLICT(title_id,session_id) DO UPDATE SET seconds=GREATEST(reader_time.seconds,EXCLUDED.seconds)`, id, input.SessionID, input.Seconds)
	}
	if err == nil && input.CurrentPage > previous {
		_, err = tx.Exec(r.Context(), `INSERT INTO reading_sessions(workspace_id,title_id,start_page,end_page,pages_read) VALUES($1,$2,$3,$4,$5)`, workspace, id, previous, input.CurrentPage, input.CurrentPage-previous)
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE book_titles SET status=CASE WHEN $1=total_pages THEN 'completed' ELSE 'reading' END,updated_at=now() WHERE id=$2`, input.CurrentPage, id)
		}
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, 500, "failed to save progress")
		return
	}
	writeJSON(w, 200, envelope{"data": envelope{"saved": true}})
}

func validReaderAnnotation(input readerAnnotation) bool {
	if input.Page < 1 || input.Page > 100000 || input.Location == "" || len(input.Location) > 2000 || len(input.Quote) > 10000 || len(input.Note) > 4000 || len(input.Rects) > 100 {
		return false
	}
	if input.Kind != "highlight" && input.Kind != "bookmark" && input.Kind != "note" {
		return false
	}
	if input.Kind == "highlight" && strings.TrimSpace(input.Quote) == "" {
		return false
	}
	if input.Kind == "note" && strings.TrimSpace(input.Note) == "" {
		return false
	}
	for _, rect := range input.Rects {
		for _, v := range []float64{rect.X, rect.Y, rect.Width, rect.Height} {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
				return false
			}
		}
		if rect.X+rect.Width > 1.001 || rect.Y+rect.Height > 1.001 {
			return false
		}
	}
	return true
}

func (api *API) getReaderAnnotations(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	rows, err := api.db.Query(r.Context(), `SELECT a.id,a.title_id,b.title,a.kind,a.location,a.page,a.quote,a.note,a.rects
 FROM reader_annotations a JOIN reader_books d ON d.title_id=a.title_id JOIN book_titles b ON b.id=a.title_id WHERE d.workspace_id=$1 ORDER BY a.created_at DESC,a.id DESC`, workspace)
	if err != nil {
		writeError(w, 500, "failed to load annotations")
		return
	}
	defer rows.Close()
	items := []envelope{}
	for rows.Next() {
		var id, titleID int64
		var title, kind, location, quote, note string
		var page int
		var rects json.RawMessage
		if rows.Scan(&id, &titleID, &title, &kind, &location, &page, &quote, &note, &rects) != nil {
			writeError(w, 500, "failed to load annotations")
			return
		}
		items = append(items, envelope{"id": id, "titleId": titleID, "title": title, "kind": kind, "location": location, "page": page, "quote": quote, "note": note, "rects": rects})
	}
	if rows.Err() != nil {
		writeError(w, 500, "failed to load annotations")
		return
	}
	writeJSON(w, 200, envelope{"data": items})
}

func (api *API) createReaderAnnotation(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	titleID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var input readerAnnotation
	if decodeJSON(r, &input) != nil || !validReaderAnnotation(input) {
		writeError(w, 422, "invalid annotation")
		return
	}
	if input.Rects == nil {
		input.Rects = []readerRect{}
	}
	rects, _ := json.Marshal(input.Rects)
	var id int64
	err = api.db.QueryRow(r.Context(), `INSERT INTO reader_annotations(title_id,kind,location,page,quote,note,rects)
 SELECT d.title_id,$1,$2,$3,$4,$5,$6 FROM reader_books d JOIN book_titles b ON b.id=d.title_id WHERE d.title_id=$7 AND d.workspace_id=$8 AND $3<=b.total_pages RETURNING id`, input.Kind, input.Location, input.Page, input.Quote, input.Note, rects, titleID, workspace).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "digital book or page not found")
		return
	}
	if err != nil {
		writeError(w, 500, "failed to save annotation")
		return
	}
	writeJSON(w, 201, envelope{"data": envelope{"id": id}})
}

func (api *API) deleteReaderAnnotation(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	result, err := api.db.Exec(r.Context(), `DELETE FROM reader_annotations a USING reader_books d WHERE a.title_id=d.title_id AND a.id=$1 AND d.workspace_id=$2`, id, workspace)
	if err != nil {
		writeError(w, 500, "failed to delete annotation")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "annotation not found")
		return
	}
	w.WriteHeader(204)
}

func (api *API) deleteReaderBook(w http.ResponseWriter, r *http.Request) {
	workspace, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, 500, "failed to resolve workspace")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	result, err := api.db.Exec(r.Context(), `DELETE FROM reader_books WHERE title_id=$1 AND workspace_id=$2`, id, workspace)
	if err != nil {
		writeError(w, 500, "failed to remove digital book")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "digital book not found")
		return
	}
	w.WriteHeader(204)
}
