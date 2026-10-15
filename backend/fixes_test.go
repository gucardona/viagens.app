package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLoginRateLimitParallel(t *testing.T) {
	app, srv := newTestApp(t)
	createUser(app.db, "ana@x.com", "Ana", "segredo123")
	var wg sync.WaitGroup
	var mu sync.Mutex
	tried := 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("POST", srv.URL+"/api/login", strings.NewReader(`{"email":"ana@x.com","password":"errada"}`))
			req.Header.Set("X-Requested-With", "viagens")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			res.Body.Close()
			if res.StatusCode == 401 {
				mu.Lock()
				tried++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if tried > loginMaxFails {
		t.Fatalf("%d password checks went through, want at most %d", tried, loginMaxFails)
	}
}

func TestRemovingPinClearsCountry(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))
	app.db.Exec(`UPDATE items SET country_code = 'pt', area = 'Alfama', geo_key = '38.71,-9.14' WHERE id = ?`, item.ID)
	noPin := hotel()
	noPin["lat"], noPin["lng"] = nil, nil
	if res := do(t, c, "PUT", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), noPin); res.StatusCode != 200 {
		t.Fatalf("put %d", res.StatusCode)
	}
	if country, area := itemGeo(t, app, item.ID); country != "" || area != "" {
		t.Fatalf("country %q area %q", country, area)
	}
}

func TestExpenseRateTooHigh(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	e := expense(ana, 100_000_000_000, "EUR")
	e["rate"] = 1e300
	if res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID), e); res.StatusCode != 400 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestRateFailureAskedOncePerCurrency(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	calls := fakeRates(t, app, 404, `{"message":"not found"}`)
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)
	for i := 0; i < 3; i++ {
		do(t, c, "POST", url, expense(ana, 1000, "ARS"))
	}
	for i := 0; i < 2; i++ {
		if s := decode[ExpenseSummary](t, do(t, c, "GET", url, nil)); s.MissingRate != 3 {
			t.Fatalf("missing %d", s.MissingRate)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("Frankfurter called %d times, want 1", n)
	}
}

func TestTodaysRateIsNotStoredForever(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	fakeRates(t, app, 200, `{"rates":{"BRL":6}}`)
	e := expense(ana, 1000, "EUR")
	e["date"] = time.Now().Format("2006-01-02")
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)
	do(t, c, "POST", url, e)
	if s := decode[ExpenseSummary](t, do(t, c, "GET", url, nil)); s.TotalCents != 6000 {
		t.Fatalf("total %d", s.TotalCents)
	}
	var n int
	app.db.QueryRow(`SELECT count(*) FROM rates`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d rates stored for today", n)
	}
}

func TestChecklistOfUnknownTrip(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	if res := do(t, c, "GET", srv.URL+"/api/trips/99999/checklist", nil); res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestDownloadMissingFile(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	att := decode[Attachment](t, upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "a.pdf", "application/pdf", []byte("%PDF"), 0))
	os.Remove(app.filePath(att.ID))
	if res := do(t, c, "GET", fmt.Sprintf("%s/api/attachments/%d", srv.URL, att.ID), nil); res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
}
