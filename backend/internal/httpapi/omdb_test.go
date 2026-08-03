package httpapi

import "testing"

func TestParseLeadingNumber(t *testing.T) {
	for input, expected := range map[string]int{"49 min": 49, "2008–2013": 2008, "5": 5, "N/A": 0} {
		if actual := parseLeadingNumber(input); actual != expected {
			t.Fatalf("parseLeadingNumber(%q) = %d, want %d", input, actual, expected)
		}
	}
}

func TestValidIMDbID(t *testing.T) {
	if !validIMDbID("tt0903747") {
		t.Fatal("expected a valid IMDb id")
	}
	for _, value := range []string{"", "nm0903747", "ttabc", "tt"} {
		if validIMDbID(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}
