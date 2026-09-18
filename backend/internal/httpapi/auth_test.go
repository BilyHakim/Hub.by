package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestSessionCookieUsesCrossSiteAttributesForHTTPS(t *testing.T) {
	api := &API{secureCookies: true}
	response := httptest.NewRecorder()
	api.setSessionCookie(response, "test-session", time.Now().Add(time.Hour))

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.Secure {
		t.Fatal("HTTPS session cookie must be Secure")
	}
	if !cookie.HttpOnly {
		t.Fatal("session cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteNoneMode {
		t.Fatalf("SameSite = %v, want None", cookie.SameSite)
	}
}

func TestSessionCookieRemainsLaxForLocalHTTP(t *testing.T) {
	api := &API{}
	response := httptest.NewRecorder()
	api.setSessionCookie(response, "test-session", time.Now().Add(time.Hour))

	cookie := response.Result().Cookies()[0]
	if cookie.Secure {
		t.Fatal("local HTTP session cookie must not be Secure")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want Lax", cookie.SameSite)
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

func TestClientIPOnlyTrustsForwardedHeadersWhenConfigured(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	request.RemoteAddr = "192.0.2.10:54321"
	request.Header.Set("X-Real-IP", "198.51.100.20")

	if got := (&API{}).clientIP(request); got != "192.0.2.10" {
		t.Fatalf("untrusted proxy client IP = %q, want remote address", got)
	}
	if got := (&API{trustProxy: true}).clientIP(request); got != "198.51.100.20" {
		t.Fatalf("trusted proxy client IP = %q, want forwarded address", got)
	}
}
