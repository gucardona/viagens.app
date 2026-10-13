package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var lisboa = map[string]string{"name": "Lisboa", "destination": "Portugal", "start_date": "2026-11-01", "end_date": "2026-11-10"}

func createTrip(t *testing.T, c *http.Client, srv *httptest.Server) Trip {
	t.Helper()
	res := do(t, c, "POST", srv.URL+"/api/trips", lisboa)
	if res.StatusCode != 201 {
		t.Fatalf("create trip: %d", res.StatusCode)
	}
	return decode[Trip](t, res)
}

func TestTripCRUD(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	if trip.ID == 0 || trip.Name != "Lisboa" || trip.EndDate != "2026-11-10" {
		t.Fatalf("created: %+v", trip)
	}

	list := decode[[]Trip](t, do(t, c, "GET", srv.URL+"/api/trips", nil))
	if len(list) != 1 || list[0].ID != trip.ID {
		t.Fatalf("list: %+v", list)
	}

	upd := map[string]string{"name": "Lisboa e Porto", "destination": "Portugal", "start_date": "2026-11-01", "end_date": "2026-11-10"}
	res := do(t, c, "PUT", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), upd)
	if got := decode[Trip](t, res); res.StatusCode != 200 || got.Name != "Lisboa e Porto" {
		t.Fatalf("update: %d %+v", res.StatusCode, got)
	}

	if res := do(t, c, "DELETE", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	if res := do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil); res.StatusCode != 404 {
		t.Fatalf("get after delete: %d", res.StatusCode)
	}
}

func TestTripValidation(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	cases := map[string]map[string]string{
		"empty name":   {"name": " ", "start_date": "2026-11-01", "end_date": "2026-11-10"},
		"bad date":     {"name": "X", "start_date": "01/11/2026", "end_date": "2026-11-10"},
		"end before":   {"name": "X", "start_date": "2026-11-10", "end_date": "2026-11-01"},
		"too long":     {"name": "X", "start_date": "2026-01-01", "end_date": "2027-01-02"},
		"missing date": {"name": "X", "start_date": "", "end_date": "2026-11-10"},
	}
	for name, body := range cases {
		if res := do(t, c, "POST", srv.URL+"/api/trips", body); res.StatusCode != 400 {
			t.Errorf("%s: %d", name, res.StatusCode)
		}
	}
}
