package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func importURL(srv string, tripID int64) string {
	return fmt.Sprintf("%s/api/trips/%d/import", srv, tripID)
}

func TestImport(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	places := []map[string]any{
		{"title": "Castelo", "notes": "vista", "lat": 38.7139, "lng": -9.1335},
		{"title": "Pastéis", "url": "https://www.google.com/maps/place/P/data=!3d38.6975!4d-9.2032"},
		{"title": "Tasca do Chico", "kind": "restaurant", "url": "https://www.google.com/maps/place/Tasca/data=!4m2!3m1!1s0x0"},
	}
	res := do(t, c, "POST", importURL(srv.URL, trip.ID), places)
	if got := decode[map[string]int](t, res); res.StatusCode != 200 || got["created"] != 3 || got["skipped"] != 0 {
		t.Fatalf("%d %v", res.StatusCode, got)
	}
	items := decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)).Items
	if len(items) != 3 || items[0].StartDate != "" || items[0].Kind != "activity" || items[0].Notes != "vista" ||
		items[1].Lat == nil || *items[1].Lat != 38.6975 || items[2].Lat != nil || items[2].Kind != "restaurant" {
		t.Fatalf("%+v", items)
	}

	// Same title without pin, and same title 30 m away, are duplicates.
	again := []map[string]any{
		{"title": "tasca do chico"},
		{"title": "Castelo", "lat": 38.7141, "lng": -9.1335},
		{"title": "Castelo", "lat": 38.80, "lng": -9.1335},
	}
	res = do(t, c, "POST", importURL(srv.URL, trip.ID), again)
	if got := decode[map[string]int](t, res); got["created"] != 1 || got["skipped"] != 2 {
		t.Fatalf("%v", got)
	}

	if res := do(t, c, "POST", importURL(srv.URL, trip.ID), []map[string]any{{"title": " "}}); res.StatusCode != 400 {
		t.Fatalf("empty title: %d", res.StatusCode)
	}
}

func TestImportNameSearch(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	app.geoGate = &rateGate{}
	var queries []string
	app.geocodeURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query().Get("q"))
		if r.URL.Query().Get("q") == "Tasca do Chico, Portugal" {
			w.Write([]byte(`[{"lat":"38.71","lon":"-9.14","category":"amenity","type":"restaurant"}]`))
			return
		}
		w.Write([]byte(`[]`))
	})
	body := `{"address":{"country_code":"pt"}}`
	fakeReverse(t, app, 200, &body)
	do(t, c, "POST", importURL(srv.URL, trip.ID), []map[string]any{{"title": "Tasca do Chico"}, {"title": "Lugar inventado"}})

	if err := app.fillGeo(context.Background()); err != nil {
		t.Fatal(err)
	}
	app.fillGeo(context.Background())
	items := decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)).Items
	if items[0].Lat == nil || *items[0].Lat != 38.71 || items[0].Kind != "restaurant" || items[0].CountryCode != "pt" {
		t.Fatalf("found: %+v", items[0])
	}
	if items[1].Lat != nil || len(queries) != 2 {
		t.Fatalf("not found: %+v queries=%v", items[1], queries)
	}
}
