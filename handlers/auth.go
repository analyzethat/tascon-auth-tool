package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookieName = "powerbi_session"
	sessionDuration   = 8 * time.Hour
)

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

var sessions = &sessionStore{
	sessions: make(map[string]time.Time),
}

func (s *sessionStore) create() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	tokenStr := base64.URLEncoding.EncodeToString(token)

	s.mu.Lock()
	s.sessions[tokenStr] = time.Now().Add(sessionDuration)
	s.mu.Unlock()

	return tokenStr, nil
}

func (s *sessionStore) valid(token string) bool {
	s.mu.RLock()
	expiry, exists := s.sessions[token]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		s.mu.Lock()
		delete(s.sessions, token)
		s.mu.Unlock()
		return false
	}

	return true
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// GetAdminPassword retrieves the admin password from environment
func GetAdminPassword() (string, bool) {
	pw := os.Getenv("POWERBI_ADMIN_PASSWORD")
	return pw, pw != ""
}

// LoginPage displays the login page
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to home
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if sessions.valid(cookie.Value) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	data := struct {
		Error string
	}{
		Error: r.URL.Query().Get("error"),
	}

	if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Login processes the login form
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	adminPassword, hasPassword := GetAdminPassword()

	// If no admin password configured, allow login
	if !hasPassword {
		h.createSessionAndRedirect(w, r)
		return
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(password), []byte(adminPassword)) != 1 {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	h.createSessionAndRedirect(w, r)
}

func (h *Handler) createSessionAndRedirect(w http.ResponseWriter, r *http.Request) {
	token, err := sessions.create()
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout logs the user out
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		sessions.delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// AuthMiddleware protects routes that require authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login page and static files
		if r.URL.Path == "/login" ||
			r.URL.Path == "/logout" ||
			strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// If no admin password configured, skip auth
		if _, hasPassword := GetAdminPassword(); !hasPassword {
			next.ServeHTTP(w, r)
			return
		}

		// Check session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !sessions.valid(cookie.Value) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
