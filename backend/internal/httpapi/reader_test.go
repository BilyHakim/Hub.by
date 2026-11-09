package httpapi

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestReaderFileValidation(t *testing.T) {
	if validReaderFile("book.pdf", []byte("%PDF-1.7\ncontent")) != "pdf" {
		t.Fatal("valid PDF rejected")
	}
	if validReaderFile("book.pdf", []byte("<script>bad</script>")) != "" {
		t.Fatal("spoofed PDF accepted")
	}
	if validReaderFile("book.epub", []byte("PKinvalid")) != "" {
		t.Fatal("invalid EPUB accepted")
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	mime, _ := archive.Create("mimetype")
	mime.Write([]byte("application/epub+zip"))
	container, _ := archive.Create("META-INF/container.xml")
	container.Write([]byte("<container/>"))
	archive.Close()
	if validReaderFile("BOOK.EPUB", buffer.Bytes()) != "epub" {
		t.Fatal("valid EPUB rejected")
	}
	if validReaderFile("book.zip", buffer.Bytes()) != "" {
		t.Fatal("unsupported extension accepted")
	}
}

func TestReaderProgressValidation(t *testing.T) {
	input := readerProgress{Location: "1", CurrentPage: 1, SessionID: "12345678-1234-1234-1234-123456789abc", Seconds: 10}
	if !validReaderProgress(input) {
		t.Fatal("valid progress rejected")
	}
	input.Seconds = -1
	if validReaderProgress(input) {
		t.Fatal("negative time accepted")
	}
	input.Seconds = 86401
	if validReaderProgress(input) {
		t.Fatal("excessive time accepted")
	}
	input.Seconds = 0
	input.Location = ""
	if validReaderProgress(input) {
		t.Fatal("missing position accepted")
	}
}

func TestReaderAnnotationValidation(t *testing.T) {
	input := readerAnnotation{Kind: "highlight", Location: "1", Page: 1, Quote: "Selected text", Rects: []readerRect{{X: .1, Y: .2, Width: .5, Height: .05}}}
	if !validReaderAnnotation(input) {
		t.Fatal("valid highlight rejected")
	}
	input.Rects[0].X = .9
	if validReaderAnnotation(input) {
		t.Fatal("out-of-page highlight accepted")
	}
	input.Rects = nil
	input.Quote = ""
	if validReaderAnnotation(input) {
		t.Fatal("empty highlight accepted")
	}
	input.Kind = "bookmark"
	if !validReaderAnnotation(input) {
		t.Fatal("bookmark requires no text")
	}
	input.Kind = "note"
	if validReaderAnnotation(input) {
		t.Fatal("empty note accepted")
	}
}
