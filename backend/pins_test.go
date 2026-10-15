package main

import (
	"fmt"
	"testing"
)

func TestPinsOfAllTrips(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	t1, t2 := createTrip(t, c, srv), createTrip(t, c, srv)
	do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, t1.ID), hotel())
	do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, t2.ID), hotel())
	do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, t2.ID), map[string]any{"kind": "note", "title": "Sem pino"})
	res := do(t, c, "GET", srv.URL+"/api/pins", nil)
	got := decode[[]Item](t, res)
	if res.StatusCode != 200 || len(got) != 2 || got[0].TripID != t1.ID || got[1].TripID != t2.ID || got[0].Lat == nil {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if res := do(t, newClient(), "GET", srv.URL+"/api/pins", nil); res.StatusCode != 401 {
		t.Fatalf("no session %d", res.StatusCode)
	}
}
