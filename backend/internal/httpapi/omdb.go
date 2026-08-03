package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const omdbBaseURL = "https://www.omdbapi.com/"

type omdbSearchResponse struct {
	Search []struct {
		Title  string `json:"Title"`
		Year   string `json:"Year"`
		IMDbID string `json:"imdbID"`
		Type   string `json:"Type"`
		Poster string `json:"Poster"`
	} `json:"Search"`
	TotalResults string `json:"totalResults"`
	Response     string `json:"Response"`
	Error        string `json:"Error"`
}

type omdbTitleResponse struct {
	Title        string `json:"Title"`
	Year         string `json:"Year"`
	Runtime      string `json:"Runtime"`
	Genre        string `json:"Genre"`
	Plot         string `json:"Plot"`
	Poster       string `json:"Poster"`
	IMDbID       string `json:"imdbID"`
	Type         string `json:"Type"`
	TotalSeasons string `json:"totalSeasons"`
	Response     string `json:"Response"`
	Error        string `json:"Error"`
}

type omdbSeasonResponse struct {
	Title    string `json:"Title"`
	Season   string `json:"Season"`
	Episodes []struct {
		Title      string `json:"Title"`
		Released   string `json:"Released"`
		Episode    string `json:"Episode"`
		IMDbRating string `json:"imdbRating"`
		IMDbID     string `json:"imdbID"`
	} `json:"Episodes"`
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

func (api *API) searchOMDbCatalog(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 || len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search query must contain 2 to 100 characters")
		return
	}
	params := url.Values{"s": {query}}
	if mediaType := r.URL.Query().Get("type"); mediaType == "movie" || mediaType == "series" {
		params.Set("type", mediaType)
	}
	var response omdbSearchResponse
	if !api.fetchOMDb(w, r, params, &response) {
		return
	}
	if response.Response != "True" {
		if strings.EqualFold(response.Error, "Movie not found!") {
			writeJSON(w, http.StatusOK, envelope{"data": envelope{"items": []envelope{}, "totalResults": 0}})
			return
		}
		writeError(w, http.StatusBadGateway, response.Error)
		return
	}
	items := make([]envelope, 0, len(response.Search))
	for _, item := range response.Search {
		if item.Type != "movie" && item.Type != "series" {
			continue
		}
		items = append(items, envelope{
			"title": item.Title, "year": item.Year, "imdbId": item.IMDbID,
			"mediaType": item.Type, "posterUrl": cleanOMDbValue(item.Poster),
		})
	}
	total, _ := strconv.Atoi(response.TotalResults)
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"items": items, "totalResults": total}})
}

func (api *API) getOMDbTitle(w http.ResponseWriter, r *http.Request) {
	imdbID := strings.TrimSpace(r.PathValue("imdbID"))
	if !validIMDbID(imdbID) {
		writeError(w, http.StatusBadRequest, "invalid IMDb id")
		return
	}
	var response omdbTitleResponse
	if !api.fetchOMDb(w, r, url.Values{"i": {imdbID}, "plot": {"short"}}, &response) {
		return
	}
	if response.Response != "True" {
		writeError(w, http.StatusNotFound, response.Error)
		return
	}
	runtime := parseLeadingNumber(response.Runtime)
	if runtime <= 0 {
		runtime = 45
		if response.Type == "movie" {
			runtime = 120
		}
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{
		"title": response.Title, "imdbId": response.IMDbID, "mediaType": response.Type,
		"releaseYear": parseLeadingNumber(response.Year), "runtimeMinutes": runtime,
		"genre": cleanOMDbValue(response.Genre), "posterUrl": cleanOMDbValue(response.Poster),
		"totalSeasons": parseLeadingNumber(response.TotalSeasons), "plot": cleanOMDbValue(response.Plot),
	}})
}

func (api *API) getOMDbSeason(w http.ResponseWriter, r *http.Request) {
	imdbID := strings.TrimSpace(r.PathValue("imdbID"))
	season, err := strconv.Atoi(r.PathValue("season"))
	if !validIMDbID(imdbID) || err != nil || season < 1 || season > 100 {
		writeError(w, http.StatusBadRequest, "invalid IMDb id or season")
		return
	}
	var response omdbSeasonResponse
	if !api.fetchOMDb(w, r, url.Values{"i": {imdbID}, "Season": {strconv.Itoa(season)}}, &response) {
		return
	}
	if response.Response != "True" {
		writeError(w, http.StatusNotFound, response.Error)
		return
	}
	episodes := make([]envelope, 0, len(response.Episodes))
	for _, item := range response.Episodes {
		episodes = append(episodes, envelope{
			"episodeNumber": parseLeadingNumber(item.Episode), "title": item.Title,
			"released": cleanOMDbValue(item.Released), "imdbId": item.IMDbID, "imdbRating": cleanOMDbValue(item.IMDbRating),
		})
	}
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"title": response.Title, "seasonNumber": season, "episodes": episodes}})
}

func (api *API) fetchOMDb(w http.ResponseWriter, r *http.Request, params url.Values, target any) bool {
	if api.omdbAPIKey == "" {
		writeError(w, http.StatusServiceUnavailable, "OMDb is not configured; set OMDB_API_KEY")
		return false
	}
	params.Set("apikey", api.omdbAPIKey)
	params.Set("r", "json")
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, omdbBaseURL+"?"+params.Encode(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create OMDb request")
		return false
	}
	response, err := api.httpClient.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "OMDb is currently unavailable")
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("OMDb returned status %d", response.StatusCode))
		return false
	}
	if json.NewDecoder(response.Body).Decode(target) != nil {
		writeError(w, http.StatusBadGateway, "OMDb returned an invalid response")
		return false
	}
	return true
}

func validIMDbID(value string) bool {
	if len(value) < 4 || len(value) > 16 || !strings.HasPrefix(value, "tt") {
		return false
	}
	_, err := strconv.ParseUint(value[2:], 10, 64)
	return err == nil
}

func parseLeadingNumber(value string) int {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r < '0' || r > '9' })
	if len(fields) == 0 {
		return 0
	}
	number, _ := strconv.Atoi(fields[0])
	return number
}

func cleanOMDbValue(value string) string {
	if value == "N/A" {
		return ""
	}
	return value
}
