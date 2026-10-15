package main

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCalendarExport(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID)
	h := hotel()
	h["ticket"] = "bought"
	h["notes"] = "Pedir quarto alto, com vista; perto do elevador\nLevar adaptador"
	do(t, c, "POST", url, h)
	do(t, c, "POST", url, map[string]any{"kind": "activity", "title": "Oceanário", "start_date": "2026-11-02",
		"end_date": "2026-11-03", "ticket": "needed"})
	do(t, c, "POST", url, map[string]any{"kind": "restaurant", "title": "Jantar", "start_date": "2026-11-02",
		"start_time": "20:00", "time_approx": true})
	do(t, c, "POST", url, map[string]any{"kind": "restaurant", "title": "Salvo sem data"})

	res := do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d/calendar.ics", srv.URL, trip.ID), nil)
	body, _ := io.ReadAll(res.Body)
	ics := string(body)
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/calendar") ||
		!strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("%d %v", res.StatusCode, res.Header)
	}
	if !strings.HasPrefix(ics, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(ics, "END:VCALENDAR\r\n") {
		t.Fatalf("envelope: %q", ics)
	}
	for _, line := range strings.Split(strings.TrimSuffix(ics, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("line longer than 75 octets: %q", line)
		}
	}
	unfolded := strings.ReplaceAll(ics, "\r\n ", "")
	for _, want := range []string{
		"SUMMARY:Hotel Alfama\r\n",
		"DTSTART:20261101T150000\r\n", "DTEND:20261105T110000\r\n",
		"LOCATION:Rua A\\, Lisboa\r\n", "GEO:38.71;-9.14\r\n",
		`Confirmação: ABC123\nIngresso comprado\nPedir quarto alto\, com vista\; perto do elevador\nLevar adaptador`,
		"SUMMARY:Oceanário\r\n", "DTSTART;VALUE=DATE:20261102\r\n", "DTEND;VALUE=DATE:20261104\r\n", "Comprar ingresso",
		"SUMMARY:Jantar\r\n", "DTSTART:20261102T200000\r\n", "DURATION:PT1H\r\n", "Horário aproximado",
	} {
		if !strings.Contains(unfolded, want) {
			t.Errorf("missing %q in\n%s", want, unfolded)
		}
	}
	if strings.Count(ics, "BEGIN:VEVENT") != 3 || strings.Contains(ics, "Salvo sem data") {
		t.Fatalf("events:\n%s", ics)
	}
}

func TestCalendarUnknownTrip(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "GET", srv.URL+"/api/trips/999/calendar.ics", nil); res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if res := do(t, newClient(), "GET", srv.URL+"/api/trips/1/calendar.ics", nil); res.StatusCode != 401 {
		t.Fatalf("no session: %d", res.StatusCode)
	}
}

func TestFoldKeepsRunesWhole(t *testing.T) {
	line := "DESCRIPTION:" + strings.Repeat("ã", 100)
	for _, l := range strings.Split(fold(line), "\r\n") {
		if len(l) > 75 || !utf8.ValidString(l) {
			t.Fatalf("bad line %q", l)
		}
	}
	if got := strings.ReplaceAll(fold(line), "\r\n ", ""); got != line {
		t.Fatalf("round trip %q", got)
	}
}
