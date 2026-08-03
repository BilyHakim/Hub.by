package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type watchTitleInput struct {
	Title          string `json:"title"`
	MediaType      string `json:"mediaType"`
	Genre          string `json:"genre"`
	ReleaseYear    int    `json:"releaseYear"`
	RuntimeMinutes int    `json:"runtimeMinutes"`
	TotalEpisodes  int    `json:"totalEpisodes"`
	IMDbID         string `json:"imdbId"`
	PosterURL      string `json:"posterUrl"`
	TotalSeasons   int    `json:"totalSeasons"`
}

type watchSessionBatchInput struct {
	TitleID         int64  `json:"titleId"`
	WatchedAt       string `json:"watchedAt"`
	DurationMinutes int    `json:"durationMinutes"`
	SeasonNumber    int    `json:"seasonNumber"`
	EpisodeFrom     int    `json:"episodeFrom"`
	EpisodeTo       int    `json:"episodeTo"`
	Notes           string `json:"notes"`
	IsBackfill      bool   `json:"isBackfill"`
}

type watchSessionInput struct {
	TitleID         int64  `json:"titleId"`
	WatchedAt       string `json:"watchedAt"`
	DurationMinutes int    `json:"durationMinutes"`
	SeasonNumber    int    `json:"seasonNumber"`
	EpisodeNumber   int    `json:"episodeNumber"`
	Notes           string `json:"notes"`
	IsBackfill      bool   `json:"isBackfill"`
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
				WHERE workspace_id=$1 AND NOT is_backfill AND watched_at >= date_trunc('month', CURRENT_DATE)::date),0)
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
		       COALESCE(last_session.season_number,0),COALESCE(last_session.episode_number,0),
		       t.imdb_id,t.poster_url,COALESCE(t.total_seasons,0)
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
		var title, mediaType, genre, status, lastWatchedAt, imdbID, posterURL string
		var releaseYear, runtimeMinutes, totalEpisodes, minutes, sessions, season, episode, totalSeasons int
		if rows.Scan(&id, &title, &mediaType, &genre, &releaseYear, &runtimeMinutes, &totalEpisodes,
			&status, &minutes, &sessions, &lastWatchedAt, &season, &episode, &imdbID, &posterURL, &totalSeasons) == nil {
			titles = append(titles, envelope{
				"id": id, "title": title, "mediaType": mediaType, "genre": genre,
				"releaseYear": releaseYear, "runtimeMinutes": runtimeMinutes, "totalEpisodes": totalEpisodes,
				"status": status, "watchedMinutes": minutes, "sessionCount": sessions,
				"lastWatchedAt": lastWatchedAt, "lastSeason": season, "lastEpisode": episode,
				"imdbId": imdbID, "posterUrl": posterURL, "totalSeasons": totalSeasons,
			})
		}
	}

	sessionRows, err := api.db.Query(r.Context(), `
		SELECT s.id,s.title_id,t.title,t.media_type,TO_CHAR(s.watched_at,'YYYY-MM-DD'),
		       s.duration_minutes,COALESCE(s.season_number,0),COALESCE(s.episode_number,0),s.notes,s.is_backfill
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
		var isBackfill bool
		if sessionRows.Scan(&id, &titleID, &title, &mediaType, &watchedAt, &duration, &season, &episode, &notes, &isBackfill) == nil {
			sessions = append(sessions, envelope{
				"id": id, "titleId": titleID, "title": title, "mediaType": mediaType,
				"watchedAt": watchedAt, "durationMinutes": duration,
				"seasonNumber": season, "episodeNumber": episode, "notes": notes, "isBackfill": isBackfill,
			})
		}
	}

	activityRows, err := api.db.Query(r.Context(), `
		SELECT TO_CHAR(day::date,'YYYY-MM-DD'),COALESCE(SUM(s.duration_minutes),0),COUNT(s.id)
		FROM generate_series(CURRENT_DATE - 6,CURRENT_DATE,INTERVAL '1 day') day
		LEFT JOIN watch_sessions s ON s.watched_at=day::date AND s.workspace_id=$1 AND NOT s.is_backfill
		GROUP BY day ORDER BY day
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch activity")
		return
	}
	defer activityRows.Close()
	dailyActivity := make([]envelope, 0, 7)
	for activityRows.Next() {
		var date string
		var minutes, sessions int64
		if activityRows.Scan(&date, &minutes, &sessions) == nil {
			dailyActivity = append(dailyActivity, envelope{"date": date, "minutes": minutes, "sessions": sessions})
		}
	}

	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"summary": envelope{
			"totalTitles": totalTitles, "watchingTitles": watchingTitles,
			"completedTitles": completedTitles, "totalMinutes": totalMinutes, "monthMinutes": monthMinutes,
		},
		"titles": titles, "recentSessions": sessions, "dailyActivity": dailyActivity,
	}})
}

func (api *API) getWatchTitleDetail(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	titleID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch title id")
		return
	}
	var title, mediaType, genre, status, imdbID, posterURL, lastWatchedAt string
	var releaseYear, runtimeMinutes, totalEpisodes, totalSeasons, watchedMinutes, sessionCount, lastSeason, lastEpisode int
	err = api.db.QueryRow(r.Context(), `
		SELECT t.title,t.media_type,t.genre,COALESCE(t.release_year,0),t.runtime_minutes,
		       COALESCE(t.total_episodes,0),COALESCE(t.total_seasons,0),t.status,t.imdb_id,t.poster_url,
		       COALESCE(stats.minutes,0),COALESCE(stats.sessions,0),
		       COALESCE(TO_CHAR(last_session.watched_at,'YYYY-MM-DD'),''),
		       COALESCE(last_session.season_number,0),COALESCE(last_session.episode_number,0)
		FROM watch_titles t
		LEFT JOIN LATERAL (SELECT SUM(duration_minutes) minutes,COUNT(*) sessions FROM watch_sessions WHERE title_id=t.id) stats ON TRUE
		LEFT JOIN LATERAL (SELECT watched_at,season_number,episode_number FROM watch_sessions WHERE title_id=t.id ORDER BY watched_at DESC,id DESC LIMIT 1) last_session ON TRUE
		WHERE t.id=$1 AND t.workspace_id=$2
	`, titleID, workspaceID).Scan(&title, &mediaType, &genre, &releaseYear, &runtimeMinutes,
		&totalEpisodes, &totalSeasons, &status, &imdbID, &posterURL, &watchedMinutes, &sessionCount,
		&lastWatchedAt, &lastSeason, &lastEpisode)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "title not found in active workspace")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load watch title")
		return
	}

	rows, err := api.db.Query(r.Context(), `
		SELECT id,TO_CHAR(watched_at,'YYYY-MM-DD'),duration_minutes,COALESCE(season_number,0),
		       COALESCE(episode_number,0),notes,is_backfill
		FROM watch_sessions WHERE workspace_id=$1 AND title_id=$2 ORDER BY watched_at DESC,id DESC LIMIT 250
	`, workspaceID, titleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load title history")
		return
	}
	defer rows.Close()
	sessions := make([]envelope, 0)
	for rows.Next() {
		var id int64
		var watchedAt, notes string
		var duration, season, episode int
		var isBackfill bool
		if rows.Scan(&id, &watchedAt, &duration, &season, &episode, &notes, &isBackfill) == nil {
			sessions = append(sessions, envelope{"id": id, "watchedAt": watchedAt, "durationMinutes": duration,
				"seasonNumber": season, "episodeNumber": episode, "notes": notes, "isBackfill": isBackfill})
		}
	}
	progressRows, err := api.db.Query(r.Context(), `
		SELECT season_number,array_agg(DISTINCT episode_number ORDER BY episode_number)
		FROM watch_sessions WHERE workspace_id=$1 AND title_id=$2 AND season_number IS NOT NULL AND episode_number IS NOT NULL
		GROUP BY season_number ORDER BY season_number
	`, workspaceID, titleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load title progress")
		return
	}
	defer progressRows.Close()
	seasons := make([]envelope, 0)
	for progressRows.Next() {
		var season int
		var episodes []int32
		if progressRows.Scan(&season, &episodes) == nil {
			seasons = append(seasons, envelope{"seasonNumber": season, "watchedEpisodes": episodes})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"title": envelope{"id": titleID, "title": title, "mediaType": mediaType, "genre": genre,
			"releaseYear": releaseYear, "runtimeMinutes": runtimeMinutes, "totalEpisodes": totalEpisodes,
			"totalSeasons": totalSeasons, "status": status, "imdbId": imdbID, "posterUrl": posterURL,
			"watchedMinutes": watchedMinutes, "sessionCount": sessionCount, "lastWatchedAt": lastWatchedAt,
			"lastSeason": lastSeason, "lastEpisode": lastEpisode},
		"sessions": sessions, "seasons": seasons,
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
		INSERT INTO watch_titles(workspace_id,title,media_type,genre,release_year,runtime_minutes,total_episodes,imdb_id,poster_url,total_seasons)
		VALUES($1,$2,$3,$4,NULLIF($5,0),$6,NULLIF($7,0),$8,$9,NULLIF($10,0)) RETURNING id
	`, workspaceID, input.Title, input.MediaType, input.Genre, input.ReleaseYear,
		input.RuntimeMinutes, input.TotalEpisodes, input.IMDbID, input.PosterURL, input.TotalSeasons).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "title is already in the watch library")
			return
		}
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
	if errors.Is(err, pgx.ErrNoRows) {
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
		INSERT INTO watch_sessions(workspace_id,title_id,watched_at,duration_minutes,season_number,episode_number,notes,is_backfill)
		VALUES($1,$2,$3,$4,NULLIF($5,0),NULLIF($6,0),$7,$8) RETURNING id
	`, workspaceID, input.TitleID, watchedAt, input.DurationMinutes,
		input.SeasonNumber, input.EpisodeNumber, input.Notes, input.IsBackfill).Scan(&id)
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
		"seasonNumber": input.SeasonNumber, "episodeNumber": input.EpisodeNumber, "notes": input.Notes, "isBackfill": input.IsBackfill,
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

func (api *API) createWatchSessionBatch(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	var input watchSessionBatchInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.Notes = strings.TrimSpace(input.Notes)
	if input.WatchedAt == "" {
		input.WatchedAt = time.Now().Format("2006-01-02")
	}
	watchedAt, dateErr := time.Parse("2006-01-02", input.WatchedAt)
	if dateErr != nil || input.TitleID <= 0 || input.DurationMinutes < 1 || input.DurationMinutes > 1440 ||
		input.SeasonNumber < 1 || input.EpisodeFrom < 1 || input.EpisodeTo < input.EpisodeFrom ||
		input.EpisodeTo-input.EpisodeFrom >= 200 || len(input.Notes) > 500 {
		writeError(w, http.StatusUnprocessableEntity, "invalid episode range")
		return
	}
	var mediaType string
	err = api.db.QueryRow(r.Context(), `SELECT media_type FROM watch_titles WHERE id=$1 AND workspace_id=$2`, input.TitleID, workspaceID).Scan(&mediaType)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "title not found in active workspace")
		return
	}
	if err != nil || mediaType != "series" {
		writeError(w, http.StatusUnprocessableEntity, "episode ranges are only available for series")
		return
	}
	tx, err := api.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record episodes")
		return
	}
	defer tx.Rollback(r.Context())
	recordedCount := int64(0)
	for episode := input.EpisodeFrom; episode <= input.EpisodeTo; episode++ {
		result, execErr := tx.Exec(r.Context(), `
			INSERT INTO watch_sessions(workspace_id,title_id,watched_at,duration_minutes,season_number,episode_number,notes,is_backfill)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (title_id,season_number,episode_number)
			WHERE season_number IS NOT NULL AND episode_number IS NOT NULL DO NOTHING
		`, workspaceID, input.TitleID, watchedAt, input.DurationMinutes, input.SeasonNumber, episode, input.Notes, input.IsBackfill)
		err = execErr
		if err == nil {
			recordedCount += result.RowsAffected()
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE watch_titles SET status='watching',updated_at=now() WHERE id=$1`, input.TitleID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		writeError(w, http.StatusInternalServerError, "failed to record episodes")
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": envelope{
		"titleId": input.TitleID, "seasonNumber": input.SeasonNumber,
		"episodeFrom": input.EpisodeFrom, "episodeTo": input.EpisodeTo,
		"recordedCount": recordedCount,
	}})
}

func (api *API) getWatchTitleProgress(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := api.currentWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve active workspace")
		return
	}
	titleID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch title id")
		return
	}
	rows, err := api.db.Query(r.Context(), `
		SELECT season_number,array_agg(DISTINCT episode_number ORDER BY episode_number)
		FROM watch_sessions
		WHERE workspace_id=$1 AND title_id=$2 AND season_number IS NOT NULL AND episode_number IS NOT NULL
		GROUP BY season_number ORDER BY season_number
	`, workspaceID, titleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load episode progress")
		return
	}
	defer rows.Close()
	seasons := make([]envelope, 0)
	for rows.Next() {
		var season int
		var episodes []int32
		if rows.Scan(&season, &episodes) == nil {
			seasons = append(seasons, envelope{"seasonNumber": season, "watchedEpisodes": episodes})
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"titleId": titleID, "seasons": seasons}})
}

func validWatchTitle(input *watchTitleInput) bool {
	input.Title = strings.TrimSpace(input.Title)
	input.MediaType = strings.TrimSpace(input.MediaType)
	input.Genre = strings.TrimSpace(input.Genre)
	input.IMDbID = strings.TrimSpace(input.IMDbID)
	input.PosterURL = strings.TrimSpace(input.PosterURL)
	if input.MediaType == "movie" {
		input.TotalEpisodes = 0
	}
	return len(input.Title) >= 1 && len(input.Title) <= 160 &&
		(input.MediaType == "movie" || input.MediaType == "series") && len(input.Genre) <= 80 &&
		(input.ReleaseYear == 0 || input.ReleaseYear >= 1888 && input.ReleaseYear <= 2200) &&
		input.RuntimeMinutes >= 1 && input.RuntimeMinutes <= 1440 &&
		(input.MediaType == "movie" || input.TotalEpisodes >= 0) &&
		(input.IMDbID == "" || validIMDbID(input.IMDbID)) && len(input.PosterURL) <= 1000 &&
		input.TotalSeasons >= 0 && input.TotalSeasons <= 100
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
		"imdbId": input.IMDbID, "posterUrl": input.PosterURL, "totalSeasons": input.TotalSeasons,
	}
}
