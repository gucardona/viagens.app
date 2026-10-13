package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func geoKey(lat, lng float64) string { return fmt.Sprintf("%.5f,%.5f", lat, lng) }

// reverse asks Nominatim for the country and neighbourhood of a point. A point
// Nominatim has nothing for (sea, desert) yields empty strings and no error.
func (a *App) reverse(ctx context.Context, lat, lng float64) (country, area string, err error) {
	a.geoGate.wait()
	params := url.Values{
		"lat": {strconv.FormatFloat(lat, 'f', -1, 64)}, "lon": {strconv.FormatFloat(lng, 'f', -1, 64)},
		"format": {"jsonv2"}, "zoom": {"14"}, "accept-language": {"pt-BR"},
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", a.reverseURL+"?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	res, err := geoClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", "", errUpstream
	}
	var body struct {
		Address map[string]string `json:"address"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", "", err
	}
	for _, k := range []string{"suburb", "neighbourhood", "city_district", "city", "town", "village"} {
		if v := body.Address[k]; v != "" {
			area = v
			break
		}
	}
	return body.Address["country_code"], area, nil
}

// fillGeo looks up country and area for every pinned item whose pin changed since
// the last lookup. Items that fail stay pending for the next round.
func (a *App) fillGeo(ctx context.Context) error {
	rows, err := a.db.QueryContext(ctx, `SELECT id, lat, lng, geo_key FROM items WHERE lat IS NOT NULL`)
	if err != nil {
		return err
	}
	type pending struct {
		id       int64
		lat, lng float64
	}
	var todo []pending
	for rows.Next() {
		var p pending
		var key string
		if err := rows.Scan(&p.id, &p.lat, &p.lng, &key); err != nil {
			rows.Close()
			return err
		}
		if key != geoKey(p.lat, p.lng) {
			todo = append(todo, p)
		}
	}
	rows.Close()
	var firstErr error
	for _, p := range todo {
		country, area, err := a.reverse(ctx, p.lat, p.lng)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		// The pin may have moved meanwhile; only write if it is still the one looked up.
		if _, err := a.db.ExecContext(ctx, `UPDATE items SET country_code = ?, area = ?, geo_key = ? WHERE id = ? AND lat = ? AND lng = ?`,
			country, area, geoKey(p.lat, p.lng), p.id, p.lat, p.lng); err != nil {
			return err
		}
	}
	return firstErr
}

// geoLoop runs fillGeo at start, every 10 minutes and whenever an item is saved.
func (a *App) geoLoop(ctx context.Context) {
	for {
		if err := a.fillGeo(ctx); err != nil {
			slog.Warn("country lookup", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-a.geoKick:
		case <-time.After(10 * time.Minute):
		}
	}
}

func (a *App) kickGeo() {
	select {
	case a.geoKick <- struct{}{}:
	default:
	}
}
