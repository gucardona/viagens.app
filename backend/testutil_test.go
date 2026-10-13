package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) (*App, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	app, err := newApp(filepath.Join(dir, "data"), false)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.routes())
	t.Cleanup(func() { srv.Close(); app.db.Close() })
	return app, srv
}

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// login creates ana@x.com / segredo123 and returns a client holding the session cookie.
func login(t *testing.T, app *App, srv *httptest.Server) *http.Client {
	t.Helper()
	if err := createUser(app.db, "ana@x.com", "Ana", "segredo123"); err != nil {
		t.Fatal(err)
	}
	c := newClient()
	res := do(t, c, "POST", srv.URL+"/api/login", map[string]string{"email": "ana@x.com", "password": "segredo123"})
	if res.StatusCode != 200 {
		t.Fatalf("login: %d", res.StatusCode)
	}
	return c
}

func do(t *testing.T, c *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Header.Set("X-Requested-With", "viagens")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
