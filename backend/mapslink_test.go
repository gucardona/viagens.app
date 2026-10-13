package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParseMapsURL(t *testing.T) {
	cases := []struct {
		in       string
		title    string
		lat, lng float64 // 0,0 = no coordinates
	}{
		{"https://www.google.com/maps/place/Past%C3%A9is+de+Bel%C3%A9m/@38.69,-9.20,17z/data=!3m1!4b1!4m6!3m5!1s0x0:0x0!8m2!3d38.6975!4d-9.2032", "Pastéis de Belém", 38.6975, -9.2032},
		{"https://www.google.com/maps/place/Castelo/@38.7139,-9.1335,17z", "Castelo", 38.7139, -9.1335},
		{"https://maps.google.com/?q=38.7,-9.1", "", 38.7, -9.1},
		{"https://www.google.com/maps?q=Torre+de+Bel%C3%A9m", "Torre de Belém", 0, 0},
		{"https://www.google.com/maps/search/?api=1&query=38.7,-9.1", "", 38.7, -9.1},
	}
	for _, c := range cases {
		u, _ := url.Parse(c.in)
		title, lat, lng := parseMapsURL(u)
		gotLat, gotLng := 0.0, 0.0
		if lat != nil {
			gotLat, gotLng = *lat, *lng
		}
		if title != c.title || gotLat != c.lat || gotLng != c.lng {
			t.Errorf("%s: got %q %v %v", c.in, title, gotLat, gotLng)
		}
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// redirects answers each request with a 302 to next[host], counting what was fetched.
func redirects(app *App, next map[string]string, fetched *[]string) {
	app.linkClient.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		*fetched = append(*fetched, r.URL.Host)
		h := http.Header{"Location": {next[r.URL.Host]}}
		return &http.Response{StatusCode: 302, Header: h, Body: http.NoBody, Request: r}, nil
	})
}

func TestMapsLinkShort(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	var fetched []string
	redirects(app, map[string]string{"maps.app.goo.gl": "https://www.google.com/maps/place/Castelo/@38.7139,-9.1335,17z"}, &fetched)
	res := do(t, c, "POST", srv.URL+"/api/maps-link", map[string]string{"url": "https://maps.app.goo.gl/abc"})
	got := decode[map[string]any](t, res)
	if res.StatusCode != 200 || got["title"] != "Castelo" || got["lat"] != 38.7139 || got["lng"] != -9.1335 {
		t.Fatalf("%d %v", res.StatusCode, got)
	}
	if len(fetched) != 1 {
		t.Fatalf("must not fetch the Google page itself: %v", fetched)
	}
}

func TestMapsLinkRejected(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	var fetched []string
	redirects(app, map[string]string{"maps.app.goo.gl": "https://evil.com/x"}, &fetched)
	for _, u := range []string{"https://maps.app.goo.gl/abc", "https://evil.com/maps/@1,2", "ftp://google.com/maps", "nada"} {
		res := do(t, c, "POST", srv.URL+"/api/maps-link", map[string]string{"url": u})
		if body := decode[map[string]string](t, res); res.StatusCode != 400 || body["error"] != "Link do Google Maps inválido" {
			t.Errorf("%s: %d %v", u, res.StatusCode, body)
		}
	}
	for _, h := range fetched {
		if h != "maps.app.goo.gl" {
			t.Fatalf("fetched %s", h)
		}
	}
}
