package main

import (
	"strings"
	"testing"
)

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run(nil, "unknown")
	if err == nil {
		t.Fatal("expected an error for an unknown migration command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
