package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
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

func (a *App) handleGeocode(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, 400, "Digite um endereço ou lugar")
		return
	}
	a.geoGate.wait()
	params := url.Values{"q": {q}, "format": {"jsonv2"}, "limit": {"5"}, "accept-language": {"pt-BR"}}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", a.geocodeURL+"?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	res, err := geoClient.Do(req)
	var raw []struct {
		DisplayName string `json:"display_name"`
		Lat, Lon    string
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
	for _, p := range raw {
		lat, err1 := strconv.ParseFloat(p.Lat, 64)
		lng, err2 := strconv.ParseFloat(p.Lon, 64)
		if err1 == nil && err2 == nil {
			out = append(out, GeoResult{Label: p.DisplayName, Lat: lat, Lng: lng})
		}
	}
	writeJSON(w, 200, out)
}
