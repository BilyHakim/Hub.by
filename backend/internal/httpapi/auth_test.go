package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuthRejectsMissingSession(t *testing.T) {
	api := &API{}
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	response := httptest.NewRecorder()

	api.requireAuth(next).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("protected handler was called without a session")
	}
}

func TestLoginAttemptLimit(t *testing.T) {
	api := &API{loginAttempts: make(map[string]loginAttempt)}
	for attempt := 0; attempt < maxLoginAttempts; attempt++ {
		if !api.allowLoginAttempt("192.0.2.1") {
			t.Fatalf("attempt %d was rejected too early", attempt+1)
		}
	}
	if api.allowLoginAttempt("192.0.2.1") {
		t.Fatal("attempt above limit was allowed")
	}
	api.clearLoginAttempts("192.0.2.1")
	if !api.allowLoginAttempt("192.0.2.1") {
		t.Fatal("successful login did not reset attempt limit")
	}
}
