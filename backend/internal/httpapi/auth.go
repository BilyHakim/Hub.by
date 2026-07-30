package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "hubby_session"
	sessionDuration   = 30 * 24 * time.Hour
	loginWindow       = 15 * time.Minute
	maxLoginAttempts  = 5
)

type authContextKey struct{}

type loginAttempt struct {
	count int
	reset time.Time
}

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func BootstrapAuth(ctx context.Context, db *pgxpool.Pool, email, initialPassword string) (bool, error) {
	var configured bool
	if err := db.QueryRow(ctx, `SELECT COALESCE(password_hash, '') <> '' FROM users WHERE id=1`).Scan(&configured); err != nil {
		return false, err
	}
	if configured || initialPassword == "" {
		return configured, nil
	}
	if len(initialPassword) < 10 {
		return false, errors.New("AUTH_INITIAL_PASSWORD must contain at least 10 characters")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return false, errors.New("AUTH_EMAIL must be a valid email address")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	result, err := db.Exec(ctx, `
		UPDATE users SET email=$1,password_hash=$2,updated_at=now()
		WHERE id=1 AND COALESCE(password_hash, '')=''
	`, email, string(hash))
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

func (api *API) login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !api.allowLoginAttempt(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan masuk; coba lagi dalam 15 menit")
		return
	}
	var input loginInput
	if decodeJSON(r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var userID int64
	var passwordHash string
	err := api.db.QueryRow(r.Context(), `
		SELECT id,COALESCE(password_hash,'') FROM users WHERE lower(email)=lower($1)
	`, input.Email).Scan(&userID, &passwordHash)
	if err != nil || passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "email atau kata sandi tidak sesuai")
		return
	}

	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	rawToken := hex.EncodeToString(token)
	tokenHash := hashSessionToken(rawToken)
	expiresAt := time.Now().Add(sessionDuration)
	_, err = api.db.Exec(r.Context(), `
		INSERT INTO auth_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)
	`, tokenHash, userID, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	_, _ = api.db.Exec(r.Context(), `DELETE FROM auth_sessions WHERE expires_at <= now()`)
	api.clearLoginAttempts(ip)
	api.setSessionCookie(w, rawToken, expiresAt)
	writeJSON(w, http.StatusOK, envelope{"data": envelope{"authenticated": true}})
}

func (api *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_, _ = api.db.Exec(r.Context(), `DELETE FROM auth_sessions WHERE token_hash=$1`, hashSessionToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0),
		HttpOnly: true, Secure: api.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (api *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var userID int64
		err = api.db.QueryRow(r.Context(), `
			SELECT user_id FROM auth_sessions WHERE token_hash=$1 AND expires_at > now()
		`, hashSessionToken(cookie.Value)).Scan(&userID)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0),
				HttpOnly: true, Secure: api.secureCookies, SameSite: http.SameSiteLaxMode,
			})
			writeError(w, http.StatusUnauthorized, "session expired")
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authenticatedUserID(ctx context.Context) int64 {
	userID, _ := ctx.Value(authContextKey{}).(int64)
	return userID
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (api *API) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt,
		MaxAge: int(sessionDuration.Seconds()), HttpOnly: true, Secure: api.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (api *API) allowLoginAttempt(ip string) bool {
	api.loginMu.Lock()
	defer api.loginMu.Unlock()
	now := time.Now()
	attempt := api.loginAttempts[ip]
	if now.After(attempt.reset) {
		attempt = loginAttempt{reset: now.Add(loginWindow)}
	}
	if attempt.count >= maxLoginAttempts {
		return false
	}
	attempt.count++
	api.loginAttempts[ip] = attempt
	return true
}

func (api *API) clearLoginAttempts(ip string) {
	api.loginMu.Lock()
	delete(api.loginAttempts, ip)
	api.loginMu.Unlock()
}
