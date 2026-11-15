package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type AIConfig struct {
	key          []byte
	allowedHosts map[string]bool
}

func NewAIConfig(encodedKey string, allowedHosts []string) (AIConfig, error) {
	config := AIConfig{allowedHosts: make(map[string]bool)}
	if encodedKey != "" {
		key, err := base64.StdEncoding.DecodeString(encodedKey)
		if err != nil || len(key) != 32 {
			return config, errors.New("AI_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
		}
		config.key = key
	}
	for _, host := range allowedHosts {
		config.allowedHosts[strings.ToLower(strings.TrimSpace(host))] = true
	}
	return config, nil
}

func (config AIConfig) seal(secret string, scope string) ([]byte, error) {
	block, err := aes.NewCipher(config.key)
	if err != nil {
		return nil, errors.New("enkripsi AI belum dikonfigurasi")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(secret), []byte(scope)), nil
}

func (config AIConfig) open(secret []byte, scope string) (string, error) {
	block, err := aes.NewCipher(config.key)
	if err != nil {
		return "", errors.New("enkripsi AI belum dikonfigurasi")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("credential tidak dapat dibuka")
	}
	if len(secret) < gcm.NonceSize() {
		return "", errors.New("credential tidak dapat dibuka")
	}
	plain, err := gcm.Open(nil, secret[:gcm.NonceSize()], secret[gcm.NonceSize():], []byte(scope))
	if err != nil {
		return "", errors.New("credential tidak dapat dibuka; periksa kunci enkripsi server")
	}
	return string(plain), nil
}

func aiSecretScope(workspace, user int64) string {
	return fmt.Sprintf("hubby-ai:%d:%d", workspace, user)
}

func (config AIConfig) validateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("base URL tidak valid")
	}
	trusted := config.allowedHosts[strings.ToLower(u.Host)]
	if u.Scheme != "https" && !(u.Scheme == "http" && trusted) {
		return nil, errors.New("gunakan HTTPS; endpoint HTTP lokal harus didaftarkan di AI_ALLOWED_HOSTS server")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !aiAllowedIP(ip, trusted) {
		return nil, errors.New("alamat endpoint AI tidak diizinkan")
	}
	return u, nil
}

func aiAllowedIP(ip netip.Addr, trusted bool) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	if trusted {
		return true
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() {
		return false
	}
	for _, prefix := range aiReservedNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

var aiReservedNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func (config AIConfig) client() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: nil, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 70 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil || len(ips) == 0 {
				return nil, errors.New("endpoint AI tidak dapat diakses")
			}
			trusted := config.allowedHosts[strings.ToLower(address)]
			// Validate every resolved address, and dial an IP directly to prevent DNS rebinding.
			for _, ip := range ips {
				if !aiAllowedIP(ip, trusted) {
					return nil, errors.New("alamat endpoint AI tidak diizinkan")
				}
			}
			_, port, _ := net.SplitHostPort(address)
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}
	return &http.Client{Transport: transport, Timeout: 80 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect endpoint AI tidak diizinkan") }}
}

type aiCredential struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	BaseURL   string `json:"baseUrl"`
	Format    string `json:"format"`
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
	ZDR       bool   `json:"zdr"`
	Enabled   bool   `json:"enabled"`
	Secret    []byte `json:"-"`
}

type aiProviderMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type aiProviderResult struct {
	Answer       string
	InputTokens  int64
	OutputTokens int64
}

func (api *API) aiProviderRequest(ctx context.Context, credential aiCredential, key, method, path string, body any) ([]byte, error) {
	if _, err := api.aiConfig.validateURL(credential.BaseURL); err != nil {
		return nil, err
	}
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(credential.BaseURL, "/")+path, bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("konfigurasi endpoint AI tidak valid")
	}
	req.Header.Set("Content-Type", "application/json")
	if credential.Format == "messages" {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if credential.Provider == "commandcode" && credential.ZDR {
		req.Header.Set("x-cmd-zdr", "1")
	}
	if credential.Provider == "commandcode" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := api.aiClient.Do(req)
	if err != nil {
		return nil, errors.New("koneksi AI gagal atau melewati batas waktu; cek endpoint dan status provider")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		messages := map[int]string{400: "format request atau model tidak didukung provider", 401: "API key ditolak provider", 403: "credential tidak memiliki akses ke model", 404: "endpoint atau model tidak ditemukan", 422: "konfigurasi atau kebijakan ZDR tidak didukung model", 429: "kuota atau batas request provider tercapai", 503: "provider sedang sibuk"}
		message := messages[res.StatusCode]
		if message == "" {
			message = "provider AI gagal memproses permintaan"
		}
		return nil, errors.New(message)
	}
	data, err = io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, errors.New("respons provider terlalu besar atau tidak dapat dibaca")
	}
	return data, nil
}

func (api *API) aiGenerate(ctx context.Context, credential aiCredential, key, system string, messages []aiProviderMessage) (aiProviderResult, error) {
	payload := envelope{"model": credential.Model}
	path := "/chat/completions"
	switch credential.Format {
	case "messages":
		path = "/messages"
		payload["system"], payload["messages"], payload["max_tokens"] = system, messages, credential.MaxTokens
	case "ollama":
		path = "/api/chat"
		payload["messages"] = append([]aiProviderMessage{{Role: "system", Content: system}}, messages...)
		payload["stream"] = false
		payload["options"] = envelope{"num_predict": credential.MaxTokens}
	default:
		payload["messages"] = append([]aiProviderMessage{{Role: "system", Content: system}}, messages...)
		payload["max_completion_tokens"] = credential.MaxTokens
	}
	data, err := api.aiProviderRequest(ctx, credential, key, "POST", path, payload)
	if err != nil {
		return aiProviderResult{}, err
	}
	var response struct {
		Choices []struct {
			Message      aiProviderMessage `json:"message"`
			FinishReason string            `json:"finish_reason"`
		} `json:"choices"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Message    aiProviderMessage `json:"message"`
		StopReason string            `json:"stop_reason"`
		DoneReason string            `json:"done_reason"`
		Usage      struct {
			Prompt     int64 `json:"prompt_tokens"`
			Completion int64 `json:"completion_tokens"`
			Input      int64 `json:"input_tokens"`
			Output     int64 `json:"output_tokens"`
		} `json:"usage"`
		PromptEvalCount int64 `json:"prompt_eval_count"`
		EvalCount       int64 `json:"eval_count"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return aiProviderResult{}, errors.New("format respons AI tidak valid")
	}
	result := aiProviderResult{}
	switch credential.Format {
	case "messages":
		if response.StopReason == "max_tokens" {
			return result, errors.New("jawaban terpotong; tingkatkan batas token di pengaturan")
		}
		for _, part := range response.Content {
			if part.Type == "text" {
				result.Answer += part.Text
			}
		}
		result.InputTokens, result.OutputTokens = response.Usage.Input, response.Usage.Output
	case "ollama":
		if response.DoneReason == "length" {
			return result, errors.New("jawaban terpotong; tingkatkan batas token di pengaturan")
		}
		result.Answer, result.InputTokens, result.OutputTokens = response.Message.Content, response.PromptEvalCount, response.EvalCount
	default:
		if len(response.Choices) != 1 {
			return result, errors.New("format respons AI tidak valid")
		}
		if response.Choices[0].FinishReason == "length" {
			return result, errors.New("jawaban terpotong; tingkatkan batas token di pengaturan")
		}
		result.Answer, result.InputTokens, result.OutputTokens = response.Choices[0].Message.Content, response.Usage.Prompt, response.Usage.Completion
	}
	result.Answer = strings.TrimSpace(result.Answer)
	if result.Answer == "" || len(result.Answer) > 64000 {
		return result, errors.New("AI tidak menghasilkan jawaban teks yang valid")
	}
	result.InputTokens, result.OutputTokens = max(0, result.InputTokens), max(0, result.OutputTokens)
	return result, nil
}
