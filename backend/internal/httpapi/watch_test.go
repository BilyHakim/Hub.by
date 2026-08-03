package httpapi

import "testing"

func TestValidWatchTitle(t *testing.T) {
	input := watchTitleInput{Title: "Severance", MediaType: "series", ReleaseYear: 2022, RuntimeMinutes: 50, TotalEpisodes: 19}
	if !validWatchTitle(&input) {
		t.Fatal("expected a valid series")
	}
	invalid := watchTitleInput{Title: "", MediaType: "movie", RuntimeMinutes: 0}
	if validWatchTitle(&invalid) {
		t.Fatal("expected an invalid movie")
	}
}

func TestMovieIgnoresEpisodeCount(t *testing.T) {
	input := watchTitleInput{Title: "Arrival", MediaType: "movie", RuntimeMinutes: 116, TotalEpisodes: 9}
	if !validWatchTitle(&input) {
		t.Fatal("expected a valid movie")
	}
	if input.TotalEpisodes != 0 {
		t.Fatalf("total episodes = %d, want 0", input.TotalEpisodes)
	}
}

func TestValidWatchSession(t *testing.T) {
	input := watchSessionInput{TitleID: 1, WatchedAt: "2026-08-03", DurationMinutes: 48, SeasonNumber: 1, EpisodeNumber: 2}
	if _, ok := validWatchSession(&input); !ok {
		t.Fatal("expected a valid watch session")
	}
	input.WatchedAt = "03-08-2026"
	if _, ok := validWatchSession(&input); ok {
		t.Fatal("expected invalid date format")
	}
}

func TestValidWatchStatus(t *testing.T) {
	for _, status := range []string{"planned", "watching", "completed", "dropped"} {
		if !validWatchStatus(status) {
			t.Fatalf("expected %q to be valid", status)
		}
	}
	if validWatchStatus("paused") {
		t.Fatal("expected paused to be invalid")
	}
}
