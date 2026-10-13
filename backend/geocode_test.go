package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func fakeNominatim(t *testing.T, h http.HandlerFunc) string {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestGeocode(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	var gotUA, gotFormat, gotQ string
	app.geocodeURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotFormat, gotQ = r.UserAgent(), r.URL.Query().Get("format"), r.URL.Query().Get("q")
		w.Write([]byte(`[{"display_name":"Rua A, Lisboa","lat":"38.7","lon":"-9.1"}]`))
	})
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=Rua+A", nil)
	got := decode[[]GeoResult](t, res)
	if res.StatusCode != 200 || len(got) != 1 || got[0] != (GeoResult{Label: "Rua A, Lisboa", Lat: 38.7, Lng: -9.1}) {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if gotUA != "viagens.app (self-hosted)" || gotFormat != "jsonv2" || gotQ != "Rua A" {
		t.Fatalf("upstream got ua=%q format=%q q=%q", gotUA, gotFormat, gotQ)
	}
}

func TestGeocodeEmptyQuery(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "GET", srv.URL+"/api/geocode?q=+", nil); res.StatusCode != 400 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestGeocodeUpstreamDown(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	app.geocodeURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=x", nil)
	if body := decode[map[string]string](t, res); res.StatusCode != 502 || body["error"] != "Busca de endereço indisponível" {
		t.Fatalf("%d %v", res.StatusCode, body)
	}
}

func TestGeocodeUpstreamSlow(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	app.geocodeURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(6 * time.Second):
		case <-r.Context().Done():
		}
	})
	start := time.Now()
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=x", nil)
	if res.StatusCode != 502 || time.Since(start) >= 6*time.Second {
		t.Fatalf("%d after %s", res.StatusCode, time.Since(start))
	}
}

func TestGeocodeRequiresSession(t *testing.T) {
	_, srv := newTestApp(t)
	if res := do(t, newClient(), "GET", srv.URL+"/api/geocode?q=x", nil); res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestBackup(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	createTrip(t, c, srv)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := backup(app.db, dest); err != nil {
		t.Fatal(err)
	}
	db, err := openDB(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	if err := db.QueryRow(`SELECT name FROM trips`).Scan(&name); err != nil || name != "Lisboa" {
		t.Fatalf("%q %v", name, err)
	}
}

func TestSPAFallback(t *testing.T) {
	h := spaHandler(fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("js")},
	})
	for path, want := range map[string]string{"/viagens/3": "<html>app</html>", "/": "<html>app</html>", "/assets/app.js": "js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 || rec.Body.String() != want {
			t.Errorf("%s: %d %q", path, rec.Code, rec.Body.String())
		}
	}
}

func TestUnknownAPIRouteIs404JSON(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "GET", srv.URL+"/api/nada", nil); res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
}
