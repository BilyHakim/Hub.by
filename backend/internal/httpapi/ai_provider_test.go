package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

func testAIConfig(t *testing.T, allowed ...string) AIConfig {
	t.Helper()
	config, err := NewAIConfig(base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")), allowed)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestAIEncryptionScopeAndTampering(t *testing.T) {
	config := testAIConfig(t)
	one, err := config.seal("private-api-key", aiSecretScope(1, 2))
	if err != nil {
		t.Fatal(err)
	}
	two, _ := config.seal("private-api-key", aiSecretScope(1, 2))
	if string(one) == string(two) || strings.Contains(string(one), "private-api-key") {
		t.Fatal("secrets must use random nonces and encrypted storage")
	}
	plain, err := config.open(one, aiSecretScope(1, 2))
	if err != nil || plain != "private-api-key" {
		t.Fatal("decryption failed")
	}
	if _, err = config.open(one, aiSecretScope(2, 2)); err == nil {
		t.Fatal("credential crossed workspace boundary")
	}
	if _, err = config.open(one, aiSecretScope(1, 3)); err == nil {
		t.Fatal("credential crossed user boundary")
	}
	one[len(one)-1] ^= 1
	if _, err = config.open(one, aiSecretScope(1, 2)); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err = config.open([]byte{1}, aiSecretScope(1, 2)); err == nil {
		t.Fatal("truncated ciphertext accepted")
	}
	if _, err = NewAIConfig("bad-key", nil); err == nil {
		t.Fatal("invalid encryption key accepted")
	}
}

func TestAIEndpointRestrictions(t *testing.T) {
	config := testAIConfig(t, "localhost:11434")
	for _, raw := range []string{"http://example.com", "https://user:secret@example.com", "https://example.com?key=secret", "https://example.com#fragment", "https://127.0.0.1", "https://10.0.0.1", "https://169.254.169.254", "https://[::ffff:127.0.0.1]"} {
		if _, err := config.validateURL(raw); err == nil {
			t.Errorf("accepted unsafe URL %s", raw)
		}
	}
	if _, err := config.validateURL("http://localhost:11434"); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{"127.0.0.1", "10.1.1.1", "169.254.169.254", "100.100.100.200", "::1", "::ffff:10.0.0.1", "0.0.0.0", "224.0.0.1"} {
		if aiAllowedIP(netip.MustParseAddr(address), false) {
			t.Errorf("accepted unsafe IP %s", address)
		}
	}
	if aiAllowedIP(netip.MustParseAddr("169.254.169.254"), true) {
		t.Fatal("metadata address must remain blocked")
	}
	if !aiAllowedIP(netip.MustParseAddr("127.0.0.1"), true) {
		t.Fatal("explicit local endpoint blocked")
	}
}

func TestAIModelDiscovery(t *testing.T) {
	for _, tc := range []struct {
		name, format, body, want string
		status                   int
	}{
		{"chat", "chat_completions", `{"data":[{"id":"chat","supported_endpoints":["/v1/chat/completions"]},{"id":"claude","supported_endpoints":["/v1/messages"]},{"id":"chat"},{"id":""},{"id":"legacy"}]}`, `"models":["chat","legacy"]`, 200},
		{"messages", "messages", `{"data":[{"id":"chat","supported_endpoints":["chat/completions"]},{"id":"claude","supported_endpoints":["messages"]}]}`, `"models":["claude"]`, 200},
		{"ollama", "ollama", `{"models":[{"name":"local"},{"name":"local"}]}`, `"models":["local"]`, 200},
		{"empty", "chat_completions", `{"data":[]}`, `"models":[]`, 200},
		{"invalid", "chat_completions", `broken`, "daftar model dari provider tidak valid", 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := "/models"
				if tc.format == "ollama" {
					path = "/api/tags"
				}
				authorized := r.Header.Get("Authorization") == "Bearer secret"
				if tc.format == "messages" {
					authorized = r.Header.Get("x-api-key") == "secret"
				}
				if r.Method != "GET" || r.URL.Path != path || !authorized {
					t.Error("incorrect model discovery request")
				}
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			config := testAIConfig(t, u.Host)
			api := &API{aiConfig: config, aiClient: config.client()}
			res := httptest.NewRecorder()
			api.writeAIModels(res, httptest.NewRequest("POST", "/", nil), aiCredential{BaseURL: server.URL, Format: tc.format}, "secret")
			if res.Code != tc.status || !strings.Contains(res.Body.String(), tc.want) || strings.Contains(res.Body.String(), "secret") {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestAIProviderFormatsAndErrors(t *testing.T) {
	for _, format := range []string{"chat_completions", "messages", "ollama"} {
		t.Run(format, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("invalid request JSON")
				}
				if body["model"] != "test-model" {
					t.Error("model missing")
				}
				switch format {
				case "messages":
					if r.URL.Path != "/messages" || r.Header.Get("x-api-key") != "secret" || body["system"] != "system" {
						t.Error("wrong Anthropic contract")
					}
					w.Write([]byte(`{"content":[{"type":"text","text":"Analisis"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
				case "ollama":
					if r.URL.Path != "/api/chat" || body["stream"] != false {
						t.Error("wrong Ollama contract")
					}
					w.Write([]byte(`{"message":{"content":"Analisis"},"prompt_eval_count":10,"eval_count":5}`))
				default:
					if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" {
						t.Error("wrong chat contract")
					}
					w.Write([]byte(`{"choices":[{"message":{"content":"Analisis"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
				}
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			config := testAIConfig(t, u.Host)
			api := &API{aiConfig: config, aiClient: config.client()}
			result, err := api.aiGenerate(context.Background(), aiCredential{BaseURL: server.URL, Format: format, Model: "test-model", MaxTokens: 1024}, "secret", "system", []aiProviderMessage{{Role: "user", Content: "question"}})
			if err != nil || result.Answer != "Analisis" || result.InputTokens != 10 || result.OutputTokens != 5 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("secret echoed upstream"))
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	config := testAIConfig(t, u.Host)
	api := &API{aiConfig: config, aiClient: config.client()}
	_, err := api.aiGenerate(context.Background(), aiCredential{BaseURL: server.URL, Format: "chat_completions"}, "secret", "", nil)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("provider error leaked or disappeared: %v", err)
	}
}

func TestAIFinanceSimulationAndHistoryPermissions(t *testing.T) {
	if got := aiPercentOf(math.MaxInt64, 50); got != math.MaxInt64/2 {
		t.Fatalf("percentage overflowed: %d", got)
	}
	if got := aiPercentOf(999, 10); got != 99 {
		t.Fatalf("rounding = %d", got)
	}
	previous := aiFinanceContext{FinanceEnabled: true, Balances: json.RawMessage(`{"assets":1}`)}
	if aiHistoryAllowed(previous, aiSettings{FinanceEnabled: true}) {
		t.Fatal("revoked balance context leaked through history")
	}
	if aiHistoryAllowed(previous, aiSettings{IncludeBalances: true}) {
		t.Fatal("revoked Finance context leaked through history")
	}
	if !aiHistoryAllowed(previous, aiSettings{FinanceEnabled: true, IncludeBalances: true}) {
		t.Fatal("authorized history was blocked")
	}
}
