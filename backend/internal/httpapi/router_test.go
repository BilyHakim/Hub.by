package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSAllowsConfiguredOriginsWithCredentials(t *testing.T) {
	allowed := []string{
		"https://bilyhakim.site",
		"http://localhost:1420",
	}
	for _, origin := range allowed {
		t.Run(origin, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
			request.Header.Set("Origin", origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodPost)
			request.Header.Set("Access-Control-Request-Headers", "content-type, x-hubby-client")
			response := httptest.NewRecorder()

			corsMiddleware("https://bilyhakim.site", []string{"http://localhost:1420"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("preflight request should not reach the next handler")
			})).ServeHTTP(response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, origin)
			}
			if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
			}
			if got := response.Header().Get("Vary"); got != "Origin" {
				t.Fatalf("Vary = %q, want Origin", got)
			}
			if got := response.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, clientVerificationHeader) {
				t.Fatalf("Access-Control-Allow-Headers = %q, want %s", got, clientVerificationHeader)
			}
		})
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	corsMiddleware("https://bilyhakim.site", []string{"http://localhost:1420"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight request should not reach the next handler")
	})).ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin %q for unlisted origin", got)
	}
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCORSBlocksCrossSiteSimplePOSTBeforeHandler(t *testing.T) {
	called := false
	request := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{"amount":1000}`))
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()

	corsMiddleware("https://bilyhakim.site", []string{"http://localhost:1420"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("unlisted cross-site POST reached the application handler")
	}
}

func TestUnsafeRequestRequiresVerificationHeaderAndJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		header      string
		wantStatus  int
		wantCalled  bool
	}{
		{name: "missing verification header", contentType: "application/json", wantStatus: http.StatusForbidden},
		{name: "simple content type", contentType: "text/plain", header: "1", wantStatus: http.StatusUnsupportedMediaType},
		{name: "valid request", contentType: "application/json; charset=utf-8", header: "1", wantStatus: http.StatusNoContent, wantCalled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			request := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{"amount":1000}`))
			request.Header.Set("Origin", "http://localhost:1420")
			request.Header.Set("Content-Type", test.contentType)
			if test.header != "" {
				request.Header.Set(clientVerificationHeader, test.header)
			}
			response := httptest.NewRecorder()

			corsMiddleware("https://bilyhakim.site", []string{"http://localhost:1420"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if called != test.wantCalled {
				t.Fatalf("handler called = %v, want %v", called, test.wantCalled)
			}
		})
	}
}
