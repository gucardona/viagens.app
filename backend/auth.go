package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 30 * 24 * time.Hour

type App struct {
	db           *sql.DB
	dataDir      string
	cookieSecure bool
	logins       *loginLimiter
	geocodeURL   string
	geoGate      *rateGate
}

func newApp(dataDir string, cookieSecure bool) (*App, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "files"), 0o755); err != nil {
		return nil, err
	}
	db, err := openDB(filepath.Join(dataDir, "viagens.db"))
	if err != nil {
		return nil, err
	}
	return &App{
		db:           db,
		dataDir:      dataDir,
		cookieSecure: cookieSecure,
		logins:       &loginLimiter{fails: map[string][]time.Time{}},
		geocodeURL:   "https://nominatim.openstreetmap.org/search",
		geoGate:      &rateGate{every: time.Second},
	}, nil
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", a.handleLogin)
	mux.HandleFunc("POST /api/logout", a.auth(a.handleLogout))
	mux.HandleFunc("GET /api/me", a.auth(a.handleMe))
	mux.HandleFunc("GET /api/trips", a.auth(a.handleListTrips))
	mux.HandleFunc("POST /api/trips", a.auth(a.handleCreateTrip))
	mux.HandleFunc("GET /api/trips/{id}", a.auth(a.handleGetTrip))
	mux.HandleFunc("PUT /api/trips/{id}", a.auth(a.handleUpdateTrip))
	mux.HandleFunc("DELETE /api/trips/{id}", a.auth(a.handleDeleteTrip))
	mux.HandleFunc("POST /api/trips/{id}/items", a.auth(a.handleCreateItem))
	mux.HandleFunc("PUT /api/items/{id}", a.auth(a.handleUpdateItem))
	mux.HandleFunc("DELETE /api/items/{id}", a.auth(a.handleDeleteItem))
	mux.HandleFunc("POST /api/trips/{id}/attachments", a.auth(a.handleUpload))
	mux.HandleFunc("GET /api/attachments/{id}", a.auth(a.handleDownload))
	mux.HandleFunc("DELETE /api/attachments/{id}", a.auth(a.handleDeleteAttachment))
	return csrf(mux)
}

// csrf rejects writes without the custom header, which other sites' forms cannot send.
func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("X-Requested-With") != "viagens" {
			writeError(w, 403, "Requisição bloqueada")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func createUser(db *sql.DB, email, name, password string) error {
	if len(password) < 8 {
		return errors.New("a senha precisa de pelo menos 8 caracteres")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users (email, name, password_hash) VALUES (?, ?, ?)`,
		strings.ToLower(strings.TrimSpace(email)), name, string(hash))
	return err
}

func hashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if a.logins.blocked(email) {
		writeError(w, 429, "Muitas tentativas. Tente de novo em alguns minutos")
		return
	}
	var id int64
	var hash string
	err := a.db.QueryRow(`SELECT id, password_hash FROM users WHERE email = ?`, email).Scan(&id, &hash)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password))
	}
	if err != nil {
		a.logins.fail(email)
		writeError(w, 401, "E-mail ou senha incorretos")
		return
	}
	b := make([]byte, 32)
	rand.Read(b)
	tok := base64.RawURLEncoding.EncodeToString(b)
	expires := time.Now().Add(sessionTTL)
	if _, err := a.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hashToken(tok), id, expires.UTC().Format(time.DateTime)); err != nil {
		serverError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: tok, Path: "/", Expires: expires,
		HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode})
	a.handleMeFor(w, id)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		a.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(204)
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	a.handleMeFor(w, userID(r))
}

func (a *App) handleMeFor(w http.ResponseWriter, id int64) {
	var email, name string
	if err := a.db.QueryRow(`SELECT email, name FROM users WHERE id = ?`, id).Scan(&email, &name); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "email": email, "name": name})
}

type ctxKey struct{}

func userID(r *http.Request) int64 { return r.Context().Value(ctxKey{}).(int64) }

func (a *App) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session")
		var id int64
		if err == nil {
			err = a.db.QueryRow(`SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > datetime('now')`,
				hashToken(c.Value)).Scan(&id)
		}
		if err != nil {
			writeError(w, 401, "Faça login para continuar")
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	}
}

// shortcut: counts live in memory and reset on restart; fine for a single process.
type loginLimiter struct {
	mu    sync.Mutex
	fails map[string][]time.Time
}

const loginWindow, loginMaxFails = 15 * time.Minute, 5

func (l *loginLimiter) recent(email string) []time.Time {
	var keep []time.Time
	for _, t := range l.fails[email] {
		if time.Since(t) < loginWindow {
			keep = append(keep, t)
		}
	}
	l.fails[email] = keep
	return keep
}

func (l *loginLimiter) blocked(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(email)) >= loginMaxFails
}

func (l *loginLimiter) fail(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fails[email] = append(l.recent(email), time.Now())
}
