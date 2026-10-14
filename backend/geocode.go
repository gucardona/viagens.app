package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rateGate spaces calls at least `every` apart (Nominatim allows 1 req/s).
type rateGate struct {
	mu    sync.Mutex
	every time.Duration
	next  time.Time
}

func (g *rateGate) wait() {
	g.mu.Lock()
	now := time.Now()
	at := now
	if g.next.After(now) {
		at = g.next
	}
	g.next = at.Add(g.every)
	g.mu.Unlock()
	time.Sleep(at.Sub(now))
}

type GeoResult struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
}

var errUpstream = errors.New("upstream error")

var geoClient = &http.Client{Timeout: 5 * time.Second}

// handleGeocode suggests places as the user types. Photon is built for search-as-you-type,
// which Nominatim's usage policy forbids; lat/lng bias results toward the trip.
func (a *App) handleGeocode(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, 400, "Digite um endereço ou lugar")
		return
	}
	params := url.Values{"q": {q}, "limit": {"6"}}
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 == nil && err2 == nil {
		params.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
		params.Set("lon", strconv.FormatFloat(lng, 'f', -1, 64))
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", a.photonURL+"?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	res, err := geoClient.Do(req)
	var raw struct {
		Features []struct {
			Geometry   struct{ Coordinates []float64 }
			Properties struct{ Name, Street, Housenumber, City, Country string }
		}
	}
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode != 200 {
			err = errUpstream
		} else {
			err = json.NewDecoder(res.Body).Decode(&raw)
		}
	}
	if err != nil {
		writeError(w, 502, "Busca de endereço indisponível")
		return
	}
	out := []GeoResult{}
	for _, f := range raw.Features {
		if len(f.Geometry.Coordinates) != 2 {
			continue
		}
		p := f.Properties
		var parts []string
		for _, s := range []string{p.Name, strings.TrimSpace(p.Street + " " + p.Housenumber), p.City, p.Country} {
			if s != "" && !slices.Contains(parts, s) {
				parts = append(parts, s)
			}
		}
		out = append(out, GeoResult{Label: strings.Join(parts, ", "), Lat: f.Geometry.Coordinates[1], Lng: f.Geometry.Coordinates[0]})
	}
	writeJSON(w, 200, out)
}
