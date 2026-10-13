package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type importPlace struct {
	Kind  string   `json:"kind"`
	Title string   `json:"title"`
	Notes string   `json:"notes"`
	URL   string   `json:"url"`
	Lat   *float64 `json:"lat"`
	Lng   *float64 `json:"lng"`
}

type known struct {
	title    string
	lat, lng *float64
}

// duplicate: same title (any case) and both without pin, or pins less than 50 m apart.
func (k known) same(p importPlace) bool {
	if !strings.EqualFold(k.title, p.Title) {
		return false
	}
	if k.lat == nil || p.Lat == nil {
		return k.lat == nil && p.Lat == nil
	}
	return distanceM(*k.lat, *k.lng, *p.Lat, *p.Lng) < 50
}

// handleImport saves places (Takeout, shared links, nearby results) as undated items of a trip.
func (a *App) handleImport(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	var places []importPlace
	if !readJSON(w, r, &places) {
		return
	}
	if len(places) > 2000 {
		writeError(w, 400, "Importe no máximo 2000 lugares por vez")
		return
	}
	for i := range places {
		p := &places[i]
		p.Title = strings.TrimSpace(p.Title)
		if p.Kind == "" {
			p.Kind = "activity"
		}
		if p.Lat == nil {
			if u, err := url.Parse(p.URL); err == nil && googleHosts[u.Host] {
				_, p.Lat, p.Lng = parseMapsURL(u)
			}
		}
		in := itemInput{Kind: p.Kind, Title: p.Title, Lat: p.Lat, Lng: p.Lng}
		if msg := in.validate(); msg != "" {
			writeError(w, 400, msg)
			return
		}
	}

	rows, err := a.db.Query(`SELECT title, lat, lng FROM items WHERE trip_id = ?`, tripID)
	if err != nil {
		serverError(w, err)
		return
	}
	var existing []known
	for rows.Next() {
		var k known
		if err := rows.Scan(&k.title, &k.lat, &k.lng); err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		existing = append(existing, k)
	}
	rows.Close()

	tx, err := a.db.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	created, skipped := 0, 0
next:
	for _, p := range places {
		for _, k := range existing {
			if k.same(p) {
				skipped++
				continue next
			}
		}
		if _, err := tx.Exec(`INSERT INTO items (trip_id, kind, title, notes, lat, lng, search_pending) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tripID, p.Kind, p.Title, p.Notes, p.Lat, p.Lng, p.Lat == nil); err != nil {
			serverError(w, err)
			return
		}
		existing = append(existing, known{p.Title, p.Lat, p.Lng})
		created++
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	a.kickGeo()
	writeJSON(w, 200, map[string]int{"created": created, "skipped": skipped})
}

var eateries = map[string]bool{"restaurant": true, "cafe": true, "fast_food": true, "bar": true, "pub": true}

// searchPending looks up imported places that came without a pin by "<title>, <trip destination>".
func (a *App) searchPending(ctx context.Context) error {
	rows, err := a.db.QueryContext(ctx, `SELECT items.id, items.title, trips.destination FROM items JOIN trips ON trips.id = items.trip_id WHERE search_pending = 1`)
	if err != nil {
		return err
	}
	type pending struct {
		id          int64
		title, dest string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.title, &p.dest); err != nil {
			rows.Close()
			return err
		}
		todo = append(todo, p)
	}
	rows.Close()
	var firstErr error
	for _, p := range todo {
		q := p.title
		if p.dest != "" {
			q += ", " + p.dest
		}
		found, err := a.searchOne(ctx, q)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if found == nil {
			_, err = a.db.ExecContext(ctx, `UPDATE items SET search_pending = 0 WHERE id = ?`, p.id)
		} else {
			_, err = a.db.ExecContext(ctx, `UPDATE items SET search_pending = 0, lat = ?, lng = ?,
				kind = CASE WHEN ? THEN 'restaurant' ELSE kind END WHERE id = ? AND lat IS NULL`,
				found.lat, found.lng, found.category == "amenity" && eateries[found.typ], p.id)
		}
		if err != nil {
			return err
		}
	}
	return firstErr
}

type searchHit struct {
	lat, lng      float64
	category, typ string
}

func (a *App) searchOne(ctx context.Context, q string) (*searchHit, error) {
	a.geoGate.wait()
	params := url.Values{"q": {q}, "format": {"jsonv2"}, "limit": {"1"}, "accept-language": {"pt-BR"}}
	req, _ := http.NewRequestWithContext(ctx, "GET", a.geocodeURL+"?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	res, err := geoClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errUpstream
	}
	var raw []struct{ Lat, Lon, Category, Type string }
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	lat, err1 := strconv.ParseFloat(raw[0].Lat, 64)
	lng, err2 := strconv.ParseFloat(raw[0].Lon, 64)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &searchHit{lat, lng, raw[0].Category, raw[0].Type}, nil
}
