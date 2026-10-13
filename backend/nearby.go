package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Nearby struct {
	Name         string  `json:"name"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Distance     int     `json:"distance"` // meters
	Kind         string  `json:"kind"`
	Cuisine      string  `json:"cuisine"`
	OpeningHours string  `json:"opening_hours"`
	Fee          string  `json:"fee"`
	Wheelchair   string  `json:"wheelchair"`
}

var nearbyFilters = map[string]string{
	"restaurant": `nwr(around:800,%s,%s)["amenity"~"^(restaurant|cafe|fast_food)$"]["name"];`,
	"toilets":    `nwr(around:800,%s,%s)["amenity"="toilets"]["access"!~"^(private|no|customers)$"];`,
}

// shortcut: entries are replaced but never evicted; a few KB per search is fine for two users, add eviction if memory grows.
type nearbyCache struct {
	mu   sync.Mutex
	data map[string]nearbyEntry
}

type nearbyEntry struct {
	at  time.Time
	out []Nearby
}

func distanceM(lat1, lng1, lat2, lng2 float64) float64 {
	rad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*rad, (lng2-lng1)*rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * 6371000 * math.Asin(math.Sqrt(h))
}

func (a *App) handleNearby(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter, ok := nearbyFilters[q.Get("what")]
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(q.Get("lng"), 64)
	if !ok || err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		writeError(w, 400, "Busca inválida")
		return
	}
	key := fmt.Sprintf("%s %.3f,%.3f", q.Get("what"), lat, lng)
	a.nearby.mu.Lock()
	e, hit := a.nearby.data[key]
	a.nearby.mu.Unlock()
	if hit && time.Since(e.at) < time.Hour {
		writeJSON(w, 200, e.out)
		return
	}

	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	query := "[out:json][timeout:10];" + fmt.Sprintf(filter, f(lat), f(lng)) + "out center 80;"
	req, _ := http.NewRequestWithContext(r.Context(), "POST", a.overpassURL, nil)
	req.URL.RawQuery = url.Values{"data": {query}}.Encode()
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	var body struct {
		Elements []struct {
			Lat, Lon float64
			Center   *struct{ Lat, Lon float64 }
			Tags     map[string]string
		}
	}
	res, err := overpassClient.Do(req)
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode != 200 {
			err = errUpstream
		} else {
			err = json.NewDecoder(res.Body).Decode(&body)
		}
	}
	if err != nil {
		writeError(w, 502, "Busca por perto indisponível agora")
		return
	}
	out := []Nearby{}
	for _, el := range body.Elements {
		pLat, pLng := el.Lat, el.Lon
		if el.Center != nil {
			pLat, pLng = el.Center.Lat, el.Center.Lon
		}
		t := el.Tags
		name := t["name"]
		if name == "" {
			name = "Banheiro público"
		}
		out = append(out, Nearby{Name: name, Lat: pLat, Lng: pLng, Distance: int(math.Round(distanceM(lat, lng, pLat, pLng))),
			Kind: t["amenity"], Cuisine: t["cuisine"], OpeningHours: t["opening_hours"], Fee: t["fee"], Wheelchair: t["wheelchair"]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Distance < out[j].Distance })
	if len(out) > 40 {
		out = out[:40]
	}
	a.nearby.mu.Lock()
	a.nearby.data[key] = nearbyEntry{time.Now(), out}
	a.nearby.mu.Unlock()
	writeJSON(w, 200, out)
}

var overpassClient = &http.Client{Timeout: 15 * time.Second}
