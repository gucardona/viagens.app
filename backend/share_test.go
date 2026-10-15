package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestShareLink(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	do(t, c, "PUT", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), map[string]string{"name": "Lisboa", "destination": "Portugal",
		"start_date": "2026-11-01", "end_date": "2026-11-10", "notes": "nota privada"})
	url := fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID)
	h := hotel()
	h["notes"] = "passaporte 123"
	do(t, c, "POST", url, h)
	do(t, c, "POST", url, map[string]any{"kind": "restaurant", "title": "Salvo sem data"})

	res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/share", srv.URL, trip.ID), nil)
	tok := decode[map[string]string](t, res)["share_token"]
	if res.StatusCode != 200 || len(tok) < 32 {
		t.Fatalf("share %d %q", res.StatusCode, tok)
	}
	if again := decode[map[string]string](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/share", srv.URL, trip.ID), nil))["share_token"]; again != tok {
		t.Fatalf("second share gave a new token %q", again)
	}
	if d := decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)); d.Trip.ShareToken != tok {
		t.Fatalf("trip share_token %q", d.Trip.ShareToken)
	}

	// Family: no session at all.
	res = do(t, newClient(), "GET", srv.URL+"/api/shared/"+tok, nil)
	got := decode[SharedTrip](t, res)
	if res.StatusCode != 200 || got.Trip.Name != "Lisboa" || len(got.Items) != 1 || got.Items[0].Title != "Hotel Alfama" {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if got.Trip.Notes != "" || got.Trip.ShareToken != "" || got.Items[0].ConfirmationCode != "" || got.Items[0].Notes != "" {
		t.Fatalf("private fields leaked: %+v", got)
	}

	if res := do(t, c, "DELETE", fmt.Sprintf("%s/api/trips/%d/share", srv.URL, trip.ID), nil); res.StatusCode != 204 {
		t.Fatalf("unshare %d", res.StatusCode)
	}
	if res := do(t, newClient(), "GET", srv.URL+"/api/shared/"+tok, nil); res.StatusCode != 404 {
		t.Fatalf("after unshare %d", res.StatusCode)
	}
}

func TestShareNeedsSessionAndTrip(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "POST", srv.URL+"/api/trips/999/share", nil); res.StatusCode != 404 {
		t.Fatalf("unknown trip %d", res.StatusCode)
	}
	if res := do(t, newClient(), "POST", srv.URL+"/api/trips/1/share", nil); res.StatusCode != 401 {
		t.Fatalf("no session %d", res.StatusCode)
	}
	for _, tok := range []string{"x", strings.Repeat("a", 32)} {
		if res := do(t, newClient(), "GET", srv.URL+"/api/shared/"+tok, nil); res.StatusCode != 404 {
			t.Fatalf("bad token %q: %d", tok, res.StatusCode)
		}
	}
}
