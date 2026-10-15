package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var ratesClient = &http.Client{Timeout: 5 * time.Second}

// rateMemo keeps answers that must not go in the rates table: today's rate, which ECB
// may still publish or revise, and failures, so one bad currency isn't asked once per expense.
// shortcut: in memory only, lost on restart (one call per currency then), fine for two people.
type rateMemo struct {
	mu sync.Mutex
	m  map[string]memoRate
}

type memoRate struct {
	brl   float64
	err   error
	until time.Time
}

// rate returns how many BRL one unit of cur was worth on date (Frankfurter, ECB data).
// Past dates are cached in the rates table; today's rate and failures only in memory for a while.
// A future date uses today's rate, so planned expenses follow the current rate day by day.
func (a *App) rate(ctx context.Context, date, cur string) (float64, error) {
	today := time.Now().Format("2006-01-02")
	if date > today {
		date = today
	}
	var brl float64
	if err := a.db.QueryRowContext(ctx, `SELECT brl FROM rates WHERE date = ? AND currency = ?`, date, cur).Scan(&brl); err == nil {
		return brl, nil
	}
	key := date + "|" + cur
	a.rates.mu.Lock()
	m, ok := a.rates.m[key]
	a.rates.mu.Unlock()
	if ok && time.Now().Before(m.until) {
		return m.brl, m.err
	}
	brl, err := a.fetchRate(ctx, date, cur)
	if err == nil && date < today {
		a.db.ExecContext(ctx, `INSERT OR REPLACE INTO rates (date, currency, brl) VALUES (?, ?, ?)`, date, cur, brl)
		return brl, nil
	}
	if ctx.Err() != nil {
		return 0, err
	}
	ttl := time.Hour
	if err != nil {
		ttl = 10 * time.Minute
	}
	a.rates.mu.Lock()
	a.rates.m[key] = memoRate{brl, err, time.Now().Add(ttl)}
	a.rates.mu.Unlock()
	return brl, err
}

func (a *App) fetchRate(ctx context.Context, date, cur string) (float64, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.ratesURL+"/"+date+"?from="+cur+"&to=BRL", nil)
	req.Header.Set("User-Agent", "viagens.app (self-hosted)")
	res, err := ratesClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	var body struct{ Rates map[string]float64 }
	if res.StatusCode != 200 {
		return 0, errUpstream
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return 0, err
	}
	brl := body.Rates["BRL"]
	if brl <= 0 {
		return 0, errUpstream
	}
	return brl, nil
}
