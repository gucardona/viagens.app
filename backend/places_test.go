package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
)

// fakeReverse serves body for every reverse lookup and counts the calls.
func fakeReverse(t *testing.T, app *App, status int, body *string) (*atomic.Int32, *url.Values) {
	var calls atomic.Int32
	var last url.Values
	app.reverseURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		last = r.URL.Query()
		w.WriteHeader(status)
		w.Write([]byte(*body))
	})
	app.geoGate = &rateGate{}
	return &calls, &last
}

func itemGeo(t *testing.T, app *App, id int64) (country, area string) {
	t.Helper()
	if err := app.db.QueryRow(`SELECT country_code, area FROM items WHERE id = ?`, id).Scan(&country, &area); err != nil {
		t.Fatal(err)
	}
	return
}

func TestFillGeo(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	body := `{"address":{"suburb":"Alfama","city":"Lisboa","country_code":"pt"}}`
	calls, last := fakeReverse(t, app, 200, &body)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))

	if err := app.fillGeo(context.Background()); err != nil {
		t.Fatal(err)
	}
	if country, area := itemGeo(t, app, item.ID); country != "pt" || area != "Alfama" {
		t.Fatalf("got %q %q", country, area)
	}
	q := *last
	if q.Get("lat") != "38.71" || q.Get("lon") != "-9.14" || q.Get("format") != "jsonv2" || q.Get("zoom") != "14" || q.Get("accept-language") != "pt-BR" {
		t.Fatalf("query %v", q)
	}

	app.fillGeo(context.Background())
	if calls.Load() != 1 {
		t.Fatalf("second round called again: %d", calls.Load())
	}

	moved := hotel()
	moved["lat"], moved["lng"] = 40.4, -3.7
	do(t, c, "PUT", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), moved)
	body = `{"address":{"city":"Madrid","country_code":"es"}}`
	app.fillGeo(context.Background())
	if country, area := itemGeo(t, app, item.ID); calls.Load() != 2 || country != "es" || area != "Madrid" {
		t.Fatalf("after move: %d %q %q", calls.Load(), country, area)
	}
}

func TestFillGeoAreaFallback(t *testing.T) {
	cases := map[string]string{
		`{"address":{"neighbourhood":"Chiado","city":"Lisboa","country_code":"pt"}}`: "Chiado",
		`{"address":{"city_district":"Centro","city":"Lisboa","country_code":"pt"}}`: "Centro",
		`{"address":{"town":"Sintra","country_code":"pt"}}`:                          "Sintra",
		`{"address":{"village":"Azenhas","country_code":"pt"}}`:                      "Azenhas",
	}
	for body, want := range cases {
		app, srv := newTestApp(t)
		c := login(t, app, srv)
		trip := createTrip(t, c, srv)
		fakeReverse(t, app, 200, &body)
		item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))
		app.fillGeo(context.Background())
		if _, area := itemGeo(t, app, item.ID); area != want {
			t.Errorf("%s: got %q want %q", body, area, want)
		}
	}
}

func TestFillGeoNoResult(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	body := `{"error":"Unable to geocode"}`
	calls, _ := fakeReverse(t, app, 200, &body)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))
	if err := app.fillGeo(context.Background()); err != nil {
		t.Fatal(err)
	}
	app.fillGeo(context.Background())
	if country, _ := itemGeo(t, app, item.ID); country != "" || calls.Load() != 1 {
		t.Fatalf("%q calls=%d", country, calls.Load())
	}
}

func TestFillGeoUpstreamDown(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	body := ``
	status := 500
	app.geoGate = &rateGate{}
	app.reverseURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
	res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel())
	if res.StatusCode != 201 {
		t.Fatalf("save must not depend on geocoding: %d", res.StatusCode)
	}
	item := decode[Item](t, res)
	if err := app.fillGeo(context.Background()); err == nil {
		t.Fatal("want error")
	}
	status, body = 200, `{"address":{"city":"Lisboa","country_code":"pt"}}`
	app.fillGeo(context.Background())
	if country, _ := itemGeo(t, app, item.ID); country != "pt" {
		t.Fatalf("retry: %q", country)
	}
}

func TestTripListCountries(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	for range 3 {
		do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel())
	}
	app.db.Exec(`UPDATE items SET country_code = 'pt'`)
	app.db.Exec(`UPDATE items SET country_code = 'es' WHERE id = (SELECT max(id) FROM items)`)
	list := decode[[]Trip](t, do(t, c, "GET", srv.URL+"/api/trips", nil))
	if len(list) != 1 || fmt.Sprint(list[0].Countries) != "[es pt]" {
		t.Fatalf("%+v", list)
	}
}
