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
	var gotUA, gotQ, gotLat, gotLon string
	app.photonURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotUA, gotQ, gotLat, gotLon = r.UserAgent(), q.Get("q"), q.Get("lat"), q.Get("lon")
		w.Write([]byte(`{"features":[
			{"geometry":{"coordinates":[-9.13,38.71]},"properties":{"name":"Castelo de São Jorge","street":"Rua de Santa Cruz do Castelo","city":"Lisboa","country":"Portugal"}},
			{"geometry":{"coordinates":[-9.14,38.72]},"properties":{"street":"Rua Augusta","housenumber":"10","city":"Lisboa","country":"Portugal"}},
			{"geometry":{"coordinates":[-9.15,38.73]},"properties":{"name":"Lisboa","city":"Lisboa","state":"Lisboa","country":"Portugal"}}
		]}`))
	})
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=castelo&lat=38.7&lng=-9.1", nil)
	got := decode[[]GeoResult](t, res)
	want := []GeoResult{
		{Label: "Castelo de São Jorge, Rua de Santa Cruz do Castelo, Lisboa, Portugal", Lat: 38.71, Lng: -9.13},
		{Label: "Rua Augusta 10, Lisboa, Portugal", Lat: 38.72, Lng: -9.14},
		{Label: "Lisboa, Portugal", Lat: 38.73, Lng: -9.15},
	}
	if res.StatusCode != 200 || len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if gotUA != "viagens.app (self-hosted)" || gotQ != "castelo" || gotLat != "38.7" || gotLon != "-9.1" {
		t.Fatalf("upstream got ua=%q q=%q lat=%q lon=%q", gotUA, gotQ, gotLat, gotLon)
	}
}

func TestGeocodeWithoutBias(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	var hasLat bool
	app.photonURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		hasLat = r.URL.Query().Has("lat")
		w.Write([]byte(`{"features":[]}`))
	})
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=x&lat=abc", nil)
	if got := decode[[]GeoResult](t, res); res.StatusCode != 200 || len(got) != 0 || hasLat {
		t.Fatalf("%d %v lat sent=%v", res.StatusCode, got, hasLat)
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
	app.photonURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	res := do(t, c, "GET", srv.URL+"/api/geocode?q=x", nil)
	if body := decode[map[string]string](t, res); res.StatusCode != 502 || body["error"] != "Busca de endereço indisponível" {
		t.Fatalf("%d %v", res.StatusCode, body)
	}
}

func TestGeocodeUpstreamSlow(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	app.photonURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestBackupOverwritesPreviousFile(t *testing.T) {
	app, _ := newTestApp(t)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := backup(app.db, dest); err != nil {
		t.Fatal(err)
	}
	if err := backup(app.db, dest); err != nil {
		t.Fatalf("second backup: %v", err)
	}
}
