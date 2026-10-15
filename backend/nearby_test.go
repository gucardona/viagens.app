package main

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func fakeOverpass(t *testing.T, app *App, status int, body string) (*atomic.Int32, *string) {
	var calls atomic.Int32
	var query string
	app.overpassURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Overpass reads a POST query only from the body, never from the URL.
		r.ParseForm()
		query = r.PostForm.Get("data")
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
	return &calls, &query
}

func TestNearbyRestaurants(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	_, query := fakeOverpass(t, app, 200, `{"elements":[
		{"type":"way","center":{"lat":38.705,"lon":-9.1},"tags":{"amenity":"restaurant","name":"Longe","cuisine":"portuguese","opening_hours":"Mo-Su 12:00-23:00"}},
		{"type":"node","lat":38.7005,"lon":-9.1,"tags":{"amenity":"cafe","name":"Perto"}}]}`)
	res := do(t, c, "GET", srv.URL+"/api/nearby?lat=38.7&lng=-9.1&what=restaurant", nil)
	got := decode[[]Nearby](t, res)
	if res.StatusCode != 200 || len(got) != 2 || got[0].Name != "Perto" || got[1].Name != "Longe" {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if got[0].Kind != "cafe" || got[0].Distance < 50 || got[0].Distance > 60 || got[1].Cuisine != "portuguese" || got[1].OpeningHours != "Mo-Su 12:00-23:00" {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(*query, "around:800,38.7,-9.1") || !strings.Contains(*query, "restaurant|cafe|fast_food") {
		t.Fatalf("query %q", *query)
	}
}

func TestNearbyToilets(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	_, query := fakeOverpass(t, app, 200, `{"elements":[{"type":"node","lat":38.7,"lon":-9.1,"tags":{"amenity":"toilets","fee":"yes","wheelchair":"yes"}}]}`)
	got := decode[[]Nearby](t, do(t, c, "GET", srv.URL+"/api/nearby?lat=38.7&lng=-9.1&what=toilets", nil))
	if len(got) != 1 || got[0].Name != "Banheiro público" || got[0].Fee != "yes" || got[0].Wheelchair != "yes" {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(*query, `"amenity"="toilets"`) {
		t.Fatalf("query %q", *query)
	}
}

func TestNearbyCache(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	calls, _ := fakeOverpass(t, app, 200, `{"elements":[]}`)
	do(t, c, "GET", srv.URL+"/api/nearby?lat=38.7001&lng=-9.1&what=toilets", nil)
	do(t, c, "GET", srv.URL+"/api/nearby?lat=38.7002&lng=-9.1&what=toilets", nil)
	if calls.Load() != 1 {
		t.Fatalf("calls %d", calls.Load())
	}
}

func TestNearbyBadInput(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	for _, q := range []string{"lat=38.7&lng=-9.1&what=x", "lat=abc&lng=-9.1&what=toilets", "lat=95&lng=-9.1&what=toilets", "lat=NaN&lng=NaN&what=toilets"} {
		if res := do(t, c, "GET", srv.URL+"/api/nearby?"+q, nil); res.StatusCode != 400 {
			t.Errorf("%s: %d", q, res.StatusCode)
		}
	}
}

func TestNearbyUpstreamDown(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	fakeOverpass(t, app, 504, ``)
	res := do(t, c, "GET", srv.URL+"/api/nearby?lat=38.7&lng=-9.1&what=toilets", nil)
	if body := decode[map[string]string](t, res); res.StatusCode != 502 || body["error"] != "Busca por perto indisponível agora" {
		t.Fatalf("%d %v", res.StatusCode, body)
	}
}
