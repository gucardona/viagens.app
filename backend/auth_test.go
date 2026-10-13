package main

import (
	"net/http"
	"testing"
)

func TestLoginOK(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)

	res := do(t, c, "GET", srv.URL+"/api/me", nil)
	if res.StatusCode != 200 {
		t.Fatalf("me: %d", res.StatusCode)
	}
	me := decode[map[string]any](t, res)
	if me["email"] != "ana@x.com" || me["name"] != "Ana" {
		t.Fatalf("me: %v", me)
	}
}

func TestLoginCookieFlags(t *testing.T) {
	app, srv := newTestApp(t)
	createUser(app.db, "ana@x.com", "Ana", "segredo123")
	res := do(t, newClient(), "POST", srv.URL+"/api/login", map[string]string{"email": "ana@x.com", "password": "segredo123"})
	var ck *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "session" {
			ck = c
		}
	}
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie: %+v", ck)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	app, srv := newTestApp(t)
	createUser(app.db, "ana@x.com", "Ana", "segredo123")
	res := do(t, newClient(), "POST", srv.URL+"/api/login", map[string]string{"email": "ana@x.com", "password": "errada"})
	if res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if body := decode[map[string]string](t, res); body["error"] != "E-mail ou senha incorretos" {
		t.Fatalf("body %v", body)
	}
}

func TestLoginRateLimit(t *testing.T) {
	app, srv := newTestApp(t)
	createUser(app.db, "ana@x.com", "Ana", "segredo123")
	c := newClient()
	for i := 0; i < 5; i++ {
		do(t, c, "POST", srv.URL+"/api/login", map[string]string{"email": "ana@x.com", "password": "errada"})
	}
	res := do(t, c, "POST", srv.URL+"/api/login", map[string]string{"email": "ANA@x.com", "password": "segredo123"})
	if res.StatusCode != 429 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestRequiresSession(t *testing.T) {
	_, srv := newTestApp(t)
	if res := do(t, newClient(), "GET", srv.URL+"/api/me", nil); res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestExpiredSession(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if _, err := app.db.Exec(`UPDATE sessions SET expires_at = datetime('now', '-1 minute')`); err != nil {
		t.Fatal(err)
	}
	if res := do(t, c, "GET", srv.URL+"/api/me", nil); res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestWriteNeedsCSRFHeader(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	req, _ := http.NewRequest("POST", srv.URL+"/api/logout", nil)
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestLogout(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "POST", srv.URL+"/api/logout", nil); res.StatusCode != 204 {
		t.Fatalf("logout %d", res.StatusCode)
	}
	if res := do(t, c, "GET", srv.URL+"/api/me", nil); res.StatusCode != 401 {
		t.Fatalf("me after logout %d", res.StatusCode)
	}
}
