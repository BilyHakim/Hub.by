package httpapi

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type watchTitleInput struct {
	Title          string `json:"title"`
	MediaType      string `json:"mediaType"`
	Genre          string `json:"genre"`
	ReleaseYear    int    `json:"releaseYear"`
	RuntimeMinutes int    `json:"runtimeMinutes"`
	TotalEpisodes  int    `json:"totalEpisodes"`
}

type watchSessionInput struct {
	TitleID         int64  `json:"titleId"`
	WatchedAt       string `json:"watchedAt"`
	DurationMinutes int    `json:"durationMinutes"`
	SeasonNumber    int    `json:"seasonNumber"`
	EpisodeNumber   int    `json:"episodeNumber"`
	Notes           string `json:"notes"`
}

func (api *API) getWatchOverview(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}

	var totalTitles, watchingTitles, completedTitles, totalMinutes, monthMinutes int64
	err = api.db.QueryRow(r.Context(), `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status='watching'),
			COUNT(*) FILTER (WHERE status='completed'),
			COALESCE((SELECT SUM(duration_minutes) FROM watch_sessions WHERE workspace_id=$1),0),
			COALESCE((SELECT SUM(duration_minutes) FROM watch_sessions
				WHERE workspace_id=$1 AND watched_at >= date_trunc('month', CURRENT_DATE)::date),0)
		FROM watch_titles WHERE workspace_id=$1
	`, workspaceID).Scan(&totalTitles, &watchingTitles, &completedTitles, &totalMinutes, &monthMinutes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch summary")
		return
	}

	rows, err := api.db.Query(r.Context(), `
		SELECT t.id,t.title,t.media_type,t.genre,COALESCE(t.release_year,0),t.runtime_minutes,
		       COALESCE(t.total_episodes,0),t.status,COALESCE(stats.minutes,0),COALESCE(stats.sessions,0),
		       COALESCE(TO_CHAR(last_session.watched_at,'YYYY-MM-DD'),''),
		       COALESCE(last_session.season_number,0),COALESCE(last_session.episode_number,0)
		FROM watch_titles t
		LEFT JOIN LATERAL (
			SELECT SUM(duration_minutes) AS minutes,COUNT(*) AS sessions
			FROM watch_sessions WHERE title_id=t.id
		) stats ON TRUE
		LEFT JOIN LATERAL (
			SELECT watched_at,season_number,episode_number
			FROM watch_sessions WHERE title_id=t.id ORDER BY watched_at DESC,id DESC LIMIT 1
		) last_session ON TRUE
		WHERE t.workspace_id=$1
		ORDER BY CASE t.status WHEN 'watching' THEN 0 WHEN 'planned' THEN 1 WHEN 'completed' THEN 2 ELSE 3 END,
		         last_session.watched_at DESC NULLS LAST,t.updated_at DESC
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch library")
		return
	}
	defer rows.Close()
	titles := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var title, mediaType, genre, status, lastWatchedAt string
		var releaseYear, runtimeMinutes, totalEpisodes, minutes, sessions, season, episode int
		if rows.Scan(&id, &title, &mediaType, &genre, &releaseYear, &runtimeMinutes, &totalEpisodes,
			&status, &minutes, &sessions, &lastWatchedAt, &season, &episode) == nil {
			titles = append(titles, envelope{
				"id": id, "title": title, "mediaType": mediaType, "genre": genre,
				"releaseYear": releaseYear, "runtimeMinutes": runtimeMinutes, "totalEpisodes": totalEpisodes,
				"status": status, "watchedMinutes": minutes, "sessionCount": sessions,
				"lastWatchedAt": lastWatchedAt, "lastSeason": season, "lastEpisode": episode,
			})
		}
	}

	sessionRows, err := api.db.Query(r.Context(), `
		SELECT s.id,s.title_id,t.title,t.media_type,TO_CHAR(s.watched_at,'YYYY-MM-DD'),
		       s.duration_minutes,COALESCE(s.season_number,0),COALESCE(s.episode_number,0),s.notes
		FROM watch_sessions s JOIN watch_titles t ON t.id=s.title_id
		WHERE s.workspace_id=$1 ORDER BY s.watched_at DESC,s.id DESC LIMIT 12
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch history")
		return
	}
	defer sessionRows.Close()
	sessions := make([]envelope, 0)
	for sessionRows.Next() {
		var id, titleID int64
		var title, mediaType, watchedAt, notes string
		var duration, season, episode int
		if sessionRows.Scan(&id, &titleID, &title, &mediaType, &watchedAt, &duration, &season, &episode, &notes) == nil {
			sessions = append(sessions, envelope{
				"id": id, "titleId": titleID, "title": title, "mediaType": mediaType,
				"watchedAt": watchedAt, "durationMinutes": duration,
				"seasonNumber": season, "episodeNumber": episode, "notes": notes,
			})
		}
	}

	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"summary": envelope{
			"totalTitles": totalTitles, "watchingTitles": watchingTitles,
			"completedTitles": completedTitles, "totalMinutes": totalMinutes, "monthMinutes": monthMinutes,
		},
		"titles": titles, "recentSessions": sessions,
	}})
}

func (api *API) createWatchTitle(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input watchTitleInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validWatchTitle(&input) {
		writeError(w, http.StatusUnprocessableEntity, "invalid movie or series values")
		return
	}
	var id int64
	err = api.db.QueryRow(r.Context(), `
		INSERT INTO watch_titles(workspace_id,title,media_type,genre,release_year,runtime_minutes,total_episodes)
		VALUES($1,$2,$3,$4,NULLIF($5,0),$6,NULLIF($7,0)) RETURNING id
	`, workspaceID, input.Title, input.MediaType, input.Genre, input.ReleaseYear,
		input.RuntimeMinutes, input.TotalEpisodes).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add title to watch library")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": watchTitleResponse(id, input)})
}

func (api *API) updateWatchTitleStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch title id")
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if decodeJSON(r, &input) != nil || !validWatchStatus(input.Status) {
		writeError(w, http.StatusUnprocessableEntity, "invalid watch status")
		return
	}
	result, err := api.db.Exec(r.Context(), `
		UPDATE watch_titles SET status=$1,updated_at=now() WHERE id=$2 AND workspace_id=$3
	`, input.Status, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update watch status")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "title not found in active workspace")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"id": id, "status": input.Status}})
}

func (api *API) deleteWatchTitle(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch title id")
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM watch_titles WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete watch title")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "title not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) createWatchSession(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input watchSessionInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	watchedAt, ok := validWatchSession(&input)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid watch session values")
		return
	}

	var mediaType, title string
	err = api.db.QueryRow(r.Context(), `
		SELECT media_type,title FROM watch_titles WHERE id=$1 AND workspace_id=$2
	`, input.TitleID, workspaceID).Scan(&mediaType, &title)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "title not found in active workspace")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch title")
		return
	}
	if mediaType == "series" && (input.SeasonNumber <= 0 || input.EpisodeNumber <= 0) {
		writeError(w, http.StatusUnprocessableEntity, "season and episode are required for a series")
		return
	}
	if mediaType == "movie" {
		input.SeasonNumber, input.EpisodeNumber = 0, 0
	}

	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record watch session")
		return
	}
	defer tx.Rollback(r.Context())
	var id int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO watch_sessions(workspace_id,title_id,watched_at,duration_minutes,season_number,episode_number,notes)
		VALUES($1,$2,$3,$4,NULLIF($5,0),NULLIF($6,0),$7) RETURNING id
	`, workspaceID, input.TitleID, watchedAt, input.DurationMinutes,
		input.SeasonNumber, input.EpisodeNumber, input.Notes).Scan(&id)
	if err == nil {
		status := "watching"
		if mediaType == "movie" {
			status = "completed"
		}
		_, err = tx.Exec(r.Context(), `UPDATE watch_titles SET status=$1,updated_at=now() WHERE id=$2`, status, input.TitleID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to record watch session")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"id": id, "titleId": input.TitleID, "title": title, "mediaType": mediaType,
		"watchedAt": input.WatchedAt, "durationMinutes": input.DurationMinutes,
		"seasonNumber": input.SeasonNumber, "episodeNumber": input.EpisodeNumber, "notes": input.Notes,
	}})
}

func (api *API) deleteWatchSession(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch session id")
		return
	}
	result, err := api.db.Exec(r.Context(), `DELETE FROM watch_sessions WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete watch session")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "watch session not found in active workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validWatchTitle(input *watchTitleInput) bool {
	input.Title = strings.TrimSpace(input.Title)
	input.MediaType = strings.TrimSpace(input.MediaType)
	input.Genre = strings.TrimSpace(input.Genre)
	if input.MediaType == "movie" {
		input.TotalEpisodes = 0
	}
	return len(input.Title) >= 1 && len(input.Title) <= 160 &&
		(input.MediaType == "movie" || input.MediaType == "series") && len(input.Genre) <= 80 &&
		(input.ReleaseYear == 0 || input.ReleaseYear >= 1888 && input.ReleaseYear <= 2200) &&
		input.RuntimeMinutes >= 1 && input.RuntimeMinutes <= 1440 &&
		(input.MediaType == "movie" || input.TotalEpisodes >= 0)
}

func validWatchSession(input *watchSessionInput) (time.Time, bool) {
	input.Notes = strings.TrimSpace(input.Notes)
	if input.WatchedAt == "" {
		input.WatchedAt = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", input.WatchedAt)
	valid := input.TitleID > 0 && input.DurationMinutes >= 1 && input.DurationMinutes <= 1440 &&
		input.SeasonNumber >= 0 && input.EpisodeNumber >= 0 && len(input.Notes) <= 500 && err == nil
	return date, valid
}

func validWatchStatus(status string) bool {
	return status == "planned" || status == "watching" || status == "completed" || status == "dropped"
}

func watchTitleResponse(id int64, input watchTitleInput) envelope {
	return envelope{
		"id": id, "title": input.Title, "mediaType": input.MediaType, "genre": input.Genre,
		"releaseYear": input.ReleaseYear, "runtimeMinutes": input.RuntimeMinutes,
		"totalEpisodes": input.TotalEpisodes, "status": "planned", "watchedMinutes": 0,
		"sessionCount": 0, "lastWatchedAt": "", "lastSeason": 0, "lastEpisode": 0,
	}
}
