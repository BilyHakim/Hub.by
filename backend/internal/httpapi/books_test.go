package httpapi

import (
	"encoding/json"
	"testing"
)

func TestValidOpenLibraryWorkID(t *testing.T) {
	for _, value := range []string{"OL27448W", "OL1W"} {
		if !validOpenLibraryWorkID(value) {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "OL27448M", "/works/OL27448W", "OLabcW"} {
		if validOpenLibraryWorkID(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func TestValidBookTitle(t *testing.T) {
	input := bookTitleInput{CatalogID: "OL27448W", Title: "The Lord of the Rings", Author: "J. R. R. Tolkien", PublishYear: 1954, TotalPages: 1178}
	if !validBookTitle(&input) {
		t.Fatal("expected a valid book")
	}
	manual := bookTitleInput{Title: "Buku lokal", Author: "Penulis", TotalPages: 240}
	if !validBookTitle(&manual) {
		t.Fatal("expected a manual book without catalog id to be valid")
	}
	input.TotalPages = 0
	if validBookTitle(&input) {
		t.Fatal("expected zero pages to be invalid")
	}
}

func TestValidBookStatus(t *testing.T) {
	for _, status := range []string{"planned", "reading", "completed", "dropped"} {
		if !validBookStatus(status) {
			t.Fatalf("expected %q to be valid", status)
		}
	}
	if validBookStatus("paused") {
		t.Fatal("expected paused to be invalid")
	}
}

func TestOpenLibraryDescription(t *testing.T) {
	for input, expected := range map[string]string{`"Plain description"`: "Plain description", `{"value":"Object description"}`: "Object description"} {
		var description openLibraryDescription
		if err := json.Unmarshal([]byte(input), &description); err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if string(description) != expected {
			t.Fatalf("description = %q, want %q", description, expected)
		}
	}
}
