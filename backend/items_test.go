package main

import (
	"fmt"
	"strings"
	"testing"
)

func hotel() map[string]any {
	return map[string]any{
		"kind": "lodging", "title": "Hotel Alfama", "start_date": "2026-11-01", "start_time": "15:00",
		"end_date": "2026-11-05", "end_time": "11:00", "address": "Rua A, Lisboa",
		"lat": 38.71, "lng": -9.14, "confirmation_code": "ABC123", "notes": "",
	}
}

func TestItemCRUD(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)

	res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel())
	if res.StatusCode != 201 {
		t.Fatalf("create: %d", res.StatusCode)
	}
	item := decode[Item](t, res)
	if item.Lat == nil || *item.Lat != 38.71 || item.ConfirmationCode != "ABC123" || item.TripID != trip.ID {
		t.Fatalf("created: %+v", item)
	}

	detail := decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil))
	if len(detail.Items) != 1 || detail.Items[0].Title != "Hotel Alfama" || detail.Attachments == nil {
		t.Fatalf("detail: %+v", detail)
	}

	upd := hotel()
	upd["title"] = "Hotel Baixa"
	upd["lat"], upd["lng"] = nil, nil
	res = do(t, c, "PUT", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), upd)
	if got := decode[Item](t, res); res.StatusCode != 200 || got.Title != "Hotel Baixa" || got.Lat != nil {
		t.Fatalf("update: %d %+v", res.StatusCode, got)
	}

	if res := do(t, c, "DELETE", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), nil); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	if res := do(t, c, "PUT", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), hotel()); res.StatusCode != 404 {
		t.Fatalf("update deleted: %d", res.StatusCode)
	}
}

func TestItemValidation(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	cases := map[string]func(m map[string]any){
		"bad kind":      func(m map[string]any) { m["kind"] = "boat" },
		"empty title":   func(m map[string]any) { m["title"] = "" },
		"long title":    func(m map[string]any) { m["title"] = strings.Repeat("a", 201) },
		"bad time":      func(m map[string]any) { m["start_time"] = "25:00" },
		"lat only":      func(m map[string]any) { m["lng"] = nil },
		"lat range":     func(m map[string]any) { m["lat"] = 91 },
		"end before":    func(m map[string]any) { m["end_date"] = "2026-10-30" },
		"time, no date": func(m map[string]any) { m["start_date"] = "" },
	}
	for name, mutate := range cases {
		body := hotel()
		mutate(body)
		if res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), body); res.StatusCode != 400 {
			t.Errorf("%s: %d", name, res.StatusCode)
		}
	}
}

func TestItemInUnknownTrip(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "POST", srv.URL+"/api/trips/999/items", hotel()); res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestDeleteTripCascadesItems(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel())
	do(t, c, "DELETE", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)
	var n int
	app.db.QueryRow(`SELECT count(*) FROM items`).Scan(&n)
	if n != 0 {
		t.Fatalf("items left: %d", n)
	}
}

func TestItemTicketAndApprox(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID)

	plain := decode[Item](t, do(t, c, "POST", url, hotel()))
	if plain.Ticket != "" || plain.TimeApprox || plain.CountryCode != "" || plain.Area != "" {
		t.Fatalf("defaults: %+v", plain)
	}

	in := hotel()
	in["ticket"], in["time_approx"] = "needed", true
	res := do(t, c, "POST", url, in)
	item := decode[Item](t, res)
	if res.StatusCode != 201 || item.Ticket != "needed" || !item.TimeApprox {
		t.Fatalf("create: %d %+v", res.StatusCode, item)
	}

	in["ticket"] = "bought"
	res = do(t, c, "PUT", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), in)
	if got := decode[Item](t, res); res.StatusCode != 200 || got.Ticket != "bought" {
		t.Fatalf("update: %d %+v", res.StatusCode, got)
	}

	in["ticket"] = "x"
	res = do(t, c, "POST", url, in)
	if body := decode[map[string]string](t, res); res.StatusCode != 400 || body["error"] != "Ingresso inválido" {
		t.Fatalf("invalid: %d %v", res.StatusCode, body)
	}
}
