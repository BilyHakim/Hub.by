package httpapi

import "testing"

func TestTMDBMediaType(t *testing.T) {
	for input, expected := range map[string]string{"movie": "movie", "tv": "series", "person": ""} {
		if actual := tmdbMediaType(input); actual != expected {
			t.Fatalf("tmdbMediaType(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestYearFromDate(t *testing.T) {
	for input, expected := range map[string]int{"2008-01-20": 2008, "": 0, "N/A": 0} {
		if actual := yearFromDate(input); actual != expected {
			t.Fatalf("yearFromDate(%q) = %d, want %d", input, actual, expected)
		}
	}
}

func TestValidCatalogID(t *testing.T) {
	for _, value := range []string{"1396", "tt0903747"} {
		if !validCatalogID(value) {
			t.Fatalf("expected %q to be a valid catalog id", value)
		}
	}
	for _, value := range []string{"", "0", "-1", "nm0903747", "ttabc"} {
		if validCatalogID(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestTMDBHelpers(t *testing.T) {
	if actual := tmdbPosterURL("/poster.jpg"); actual != "https://image.tmdb.org/t/p/w500/poster.jpg" {
		t.Fatalf("unexpected poster URL: %q", actual)
	}
	genres := tmdbGenreNames([]tmdbGenre{{Name: "Drama"}, {Name: "Crime"}})
	if genres != "Drama, Crime" {
		t.Fatalf("unexpected genres: %q", genres)
	}
	if runtime := firstPositive(0, 49, 50); runtime != 49 {
		t.Fatalf("runtime = %d, want 49", runtime)
	}
}
