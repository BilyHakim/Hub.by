package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	tmdbBaseURL  = "https://api.themoviedb.org/3"
	tmdbImageURL = "https://image.tmdb.org/t/p/w500"
)

type tmdbSearchResponse struct {
	Results []struct {
		ID           int64  `json:"id"`
		MediaType    string `json:"media_type"`
		Title        string `json:"title"`
		Name         string `json:"name"`
		ReleaseDate  string `json:"release_date"`
		FirstAirDate string `json:"first_air_date"`
		PosterPath   string `json:"poster_path"`
	} `json:"results"`
}

type tmdbGenre struct {
	Name string `json:"name"`
}

type tmdbTitleResponse struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	Name             string `json:"name"`
	ReleaseDate      string `json:"release_date"`
	FirstAirDate     string `json:"first_air_date"`
	Runtime          int    `json:"runtime"`
	EpisodeRunTime   []int  `json:"episode_run_time"`
	LastEpisodeToAir *struct {
		Runtime int `json:"runtime"`
	} `json:"last_episode_to_air"`
	Genres          []tmdbGenre `json:"genres"`
	Overview        string      `json:"overview"`
	PosterPath      string      `json:"poster_path"`
	NumberOfSeasons int         `json:"number_of_seasons"`
}

type tmdbSeasonResponse struct {
	Name     string `json:"name"`
	Episodes []struct {
		ID            int64   `json:"id"`
		Name          string  `json:"name"`
		AirDate       string  `json:"air_date"`
		EpisodeNumber int     `json:"episode_number"`
		VoteAverage   float64 `json:"vote_average"`
	} `json:"episodes"`
}

type tmdbFindResponse struct {
	MovieResults []struct {
		ID int64 `json:"id"`
	} `json:"movie_results"`
	TVResults []struct {
		ID int64 `json:"id"`
	} `json:"tv_results"`
}

func (api *API) searchTMDBCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 || len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search query must contain 2 to 100 characters")
		return
	}
	requestedType := r.URL.Query().Get("type")
	if requestedType != "" && requestedType != "movie" && requestedType != "series" {
		writeError(w, http.StatusBadRequest, "invalid media type")
		return
	}
	params := url.Values{
		"query":         {query},
		"include_adult": {"false"},
		"language":      {"id-ID"},
	}
	var response tmdbSearchResponse
	if !api.fetchTMDB(w, r, "/search/multi", params, &response) {
		return
	}
	items := make([]envelope, 0, len(response.Results))
	for _, item := range response.Results {
		mediaType := tmdbMediaType(item.MediaType)
		if mediaType == "" || requestedType != "" && mediaType != requestedType {
			continue
		}
		title, date := item.Title, item.ReleaseDate
		if mediaType == "series" {
			title, date = item.Name, item.FirstAirDate
		}
		items = append(items, envelope{
			"title": title, "year": yearFromDate(date), "catalogId": strconv.FormatInt(item.ID, 10),
			"mediaType": mediaType, "posterUrl": tmdbPosterURL(item.PosterPath),
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"items": items, "totalResults": len(items)}})
}

func (api *API) getTMDBTitle(w http.ResponseWriter, r *http.Request) {
	catalogID := strings.TrimSpace(r.PathValue("catalogID"))
	mediaType := r.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "series" {
		writeError(w, http.StatusBadRequest, "media type must be movie or series")
		return
	}
	tmdbID, ok := api.resolveTMDBID(w, r, catalogID, mediaType)
	if !ok {
		return
	}
	pathType := "movie"
	if mediaType == "series" {
		pathType = "tv"
	}
	var response tmdbTitleResponse
	if !api.fetchTMDB(w, r, "/"+pathType+"/"+strconv.FormatInt(tmdbID, 10), url.Values{"language": {"id-ID"}}, &response) {
		return
	}
	runtime := response.Runtime
	if mediaType == "series" {
		runtime = firstPositive(response.EpisodeRunTime...)
		if runtime == 0 && response.LastEpisodeToAir != nil {
			runtime = response.LastEpisodeToAir.Runtime
		}
	}
	if runtime <= 0 {
		runtime = 120
		if mediaType == "series" {
			runtime = 45
		}
	}
	title, date := response.Title, response.ReleaseDate
	if mediaType == "series" {
		title, date = response.Name, response.FirstAirDate
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"title": title, "catalogId": strconv.FormatInt(response.ID, 10), "mediaType": mediaType,
		"releaseYear": yearFromDate(date), "runtimeMinutes": runtime,
		"genre": tmdbGenreNames(response.Genres), "posterUrl": tmdbPosterURL(response.PosterPath),
		"totalSeasons": response.NumberOfSeasons, "plot": response.Overview,
	}})
}

func (api *API) getTMDBSeason(w http.ResponseWriter, r *http.Request) {
	catalogID := strings.TrimSpace(r.PathValue("catalogID"))
	season, err := strconv.Atoi(r.PathValue("season"))
	if err != nil || season < 1 || season > 100 {
		writeError(w, http.StatusBadRequest, "invalid season")
		return
	}
	tmdbID, ok := api.resolveTMDBID(w, r, catalogID, "series")
	if !ok {
		return
	}
	var response tmdbSeasonResponse
	path := fmt.Sprintf("/tv/%d/season/%d", tmdbID, season)
	if !api.fetchTMDB(w, r, path, url.Values{"language": {"id-ID"}}, &response) {
		return
	}
	episodes := make([]envelope, 0, len(response.Episodes))
	for _, item := range response.Episodes {
		rating := ""
		if item.VoteAverage > 0 {
			rating = strconv.FormatFloat(item.VoteAverage, 'f', 1, 64)
		}
		episodes = append(episodes, envelope{
			"episodeNumber": item.EpisodeNumber, "title": item.Name,
			"released": item.AirDate, "catalogId": strconv.FormatInt(item.ID, 10), "rating": rating,
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"title": response.Name, "seasonNumber": season, "episodes": episodes}})
}

func (api *API) resolveTMDBID(w http.ResponseWriter, r *http.Request, catalogID, mediaType string) (int64, bool) {
	if id, err := strconv.ParseInt(catalogID, 10, 64); err == nil && id > 0 {
		return id, true
	}
	if !validIMDbID(catalogID) {
		writeError(w, http.StatusBadRequest, "invalid catalog id")
		return 0, false
	}
	var response tmdbFindResponse
	params := url.Values{"external_source": {"imdb_id"}, "language": {"id-ID"}}
	if !api.fetchTMDB(w, r, "/find/"+url.PathEscape(catalogID), params, &response) {
		return 0, false
	}
	if mediaType == "movie" && len(response.MovieResults) > 0 {
		return response.MovieResults[0].ID, true
	}
	if mediaType == "series" && len(response.TVResults) > 0 {
		return response.TVResults[0].ID, true
	}
	writeError(w, http.StatusNotFound, "title was not found on TMDB")
	return 0, false
}

func (api *API) fetchTMDB(w http.ResponseWriter, r *http.Request, path string, params url.Values, target any) bool {
	if api.tmdbAPIToken == "" {
		writeError(w, http.StatusServiceUnavailable, "TMDB is not configured; set TMDB_API_TOKEN")
		return false
	}
	requestURL := tmdbBaseURL + path
	if encoded := params.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, requestURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create TMDB request")
		return false
	}
	request.Header.Set("Authorization", "Bearer "+api.tmdbAPIToken)
	request.Header.Set("Accept", "application/json")
	response, err := api.httpClient.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "TMDB is currently unavailable")
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		writeError(w, http.StatusNotFound, "title was not found on TMDB")
		return false
	}
	if response.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("TMDB returned status %d", response.StatusCode))
		return false
	}
	if json.NewDecoder(response.Body).Decode(target) != nil {
		writeError(w, http.StatusBadGateway, "TMDB returned an invalid response")
		return false
	}
	return true
}

func tmdbMediaType(value string) string {
	if value == "movie" {
		return "movie"
	}
	if value == "tv" {
		return "series"
	}
	return ""
}

func yearFromDate(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(value[:4])
	return year
}

func tmdbPosterURL(path string) string {
	if path == "" {
		return ""
	}
	return tmdbImageURL + path
}

func tmdbGenreNames(genres []tmdbGenre) string {
	names := make([]string, 0, len(genres))
	for _, genre := range genres {
		if name := strings.TrimSpace(genre.Name); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func validIMDbID(value string) bool {
	if len(value) < 4 || len(value) > 16 || !strings.HasPrefix(value, "tt") {
		return false
	}
	_, err := strconv.ParseUint(value[2:], 10, 64)
	return err == nil
}

func validCatalogID(value string) bool {
	if validIMDbID(value) {
		return true
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return err == nil && id > 0
}
