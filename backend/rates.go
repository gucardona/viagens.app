package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

var ratesClient = &http.Client{Timeout: 5 * time.Second}

// rate returns how many BRL one unit of cur was worth on date (Frankfurter, ECB data), cached in the rates table.
// A future date uses today's rate, so planned expenses follow the current rate day by day.
func (a *App) rate(ctx context.Context, date, cur string) (float64, error) {
	if today := time.Now().Format("2006-01-02"); date > today {
		date = today
	}
	var brl float64
	if err := a.db.QueryRowContext(ctx, `SELECT brl FROM rates WHERE date = ? AND currency = ?`, date, cur).Scan(&brl); err == nil {
		return brl, nil
	}
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
	brl = body.Rates["BRL"]
	if brl <= 0 {
		return 0, errUpstream
	}
	a.db.ExecContext(ctx, `INSERT OR REPLACE INTO rates (date, currency, brl) VALUES (?, ?, ?)`, date, cur, brl)
	return brl, nil
}
