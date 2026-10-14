package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

func twoUsers(t *testing.T, app *App) (ana, bruno int64) {
	t.Helper()
	if err := createUser(app.db, "bruno@x.com", "Bruno", "segredo123"); err != nil {
		t.Fatal(err)
	}
	app.db.QueryRow(`SELECT id FROM users WHERE email = 'ana@x.com'`).Scan(&ana)
	app.db.QueryRow(`SELECT id FROM users WHERE email = 'bruno@x.com'`).Scan(&bruno)
	return
}

func expense(paidBy int64, cents int64, cur string) map[string]any {
	return map[string]any{"title": "Jantar", "amount_cents": cents, "currency": cur, "category": "food",
		"paid_by": paidBy, "payer_share": 50, "date": "2026-11-02"}
}

func fakeRates(t *testing.T, app *App, status int, body string) *atomic.Int32 {
	var calls atomic.Int32
	app.ratesURL = fakeNominatim(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
	return &calls
}

func TestExpenseCRUD(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)

	res := do(t, c, "POST", url, expense(ana, 5000, "BRL"))
	e := decode[Expense](t, res)
	if res.StatusCode != 201 || e.AmountCents != 5000 || e.PaidBy != ana || e.PayerShare != 50 || e.TripID != trip.ID {
		t.Fatalf("%d %+v", res.StatusCode, e)
	}
	upd := expense(ana, 7000, "BRL")
	upd["payer_share"] = 100
	res = do(t, c, "PUT", fmt.Sprintf("%s/api/expenses/%d", srv.URL, e.ID), upd)
	if got := decode[Expense](t, res); res.StatusCode != 200 || got.AmountCents != 7000 || got.PayerShare != 100 {
		t.Fatalf("%d %+v", res.StatusCode, got)
	}
	if res := do(t, c, "DELETE", fmt.Sprintf("%s/api/expenses/%d", srv.URL, e.ID), nil); res.StatusCode != 204 {
		t.Fatalf("delete %d", res.StatusCode)
	}

	users := decode[[]map[string]any](t, do(t, c, "GET", srv.URL+"/api/users", nil))
	if len(users) != 2 || users[0]["name"] != "Ana" || users[1]["name"] != "Bruno" {
		t.Fatalf("users %v", users)
	}
}

func TestExpenseValidation(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	other := createTrip(t, c, srv)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, other.ID), hotel()))
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)
	bad := []func(m map[string]any){
		func(m map[string]any) { m["amount_cents"] = 0 },
		func(m map[string]any) { m["currency"] = "eu" },
		func(m map[string]any) { m["category"] = "x" },
		func(m map[string]any) { m["payer_share"] = 101 },
		func(m map[string]any) { m["paid_by"] = 999 },
		func(m map[string]any) { m["date"] = "ontem" },
		func(m map[string]any) { m["title"] = " " },
		func(m map[string]any) { m["item_id"] = item.ID },
		func(m map[string]any) { m["rate"] = 0 },
	}
	for k, f := range bad {
		m := expense(ana, 100, "EUR")
		f(m)
		if res := do(t, c, "POST", url, m); res.StatusCode != 400 {
			t.Errorf("case %d: %d", k, res.StatusCode)
		}
	}
}

func TestExpenseSummary(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, bruno := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)
	calls := fakeRates(t, app, 200, `{"amount":1,"base":"EUR","date":"2026-11-02","rates":{"BRL":6.0}}`)

	do(t, c, "POST", url, expense(ana, 10000, "BRL"))  // R$100, half each
	do(t, c, "POST", url, expense(bruno, 1000, "EUR")) // €10 = R$60, half each
	manual := expense(ana, 1000, "ARS")
	manual["rate"], manual["category"], manual["date"] = 0.005, "shopping", "2026-11-03"
	do(t, c, "POST", url, manual) // R$0,05 → 5 cents, half each

	s := decode[ExpenseSummary](t, do(t, c, "GET", url, nil))
	if s.TotalCents != 16005 || s.ByCategory["food"] != 16000 || s.ByCategory["shopping"] != 5 || s.ByDay["2026-11-03"] != 5 || s.MissingRate != 0 {
		t.Fatalf("%+v", s)
	}
	if len(s.Expenses) != 3 || s.Expenses[1].BRLCents == nil || *s.Expenses[1].BRLCents != 6000 {
		t.Fatalf("%+v", s.Expenses)
	}
	// Bruno owes half of Ana's R$100,05 (50,025) and Ana owes half of Bruno's R$60 (30): Bruno owes 20,025 → 20,03.
	if s.Balance == nil || s.Balance.From != "Bruno" || s.Balance.To != "Ana" || s.Balance.Cents != 2003 {
		t.Fatalf("balance %+v", s.Balance)
	}
	decode[ExpenseSummary](t, do(t, c, "GET", url, nil))
	if calls.Load() != 1 {
		t.Fatalf("rate fetched %d times", calls.Load())
	}
}

func TestExpenseSummaryNoRate(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	url := fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID)
	fakeRates(t, app, 500, ``)
	full := expense(ana, 5000, "BRL")
	full["payer_share"] = 100
	do(t, c, "POST", url, full)
	do(t, c, "POST", url, expense(ana, 1000, "EUR"))
	s := decode[ExpenseSummary](t, do(t, c, "GET", url, nil))
	if s.TotalCents != 5000 || s.MissingRate != 1 || s.Expenses[1].BRLCents != nil || s.Balance != nil {
		t.Fatalf("%+v %+v", s, s.Balance)
	}
}

func TestDeleteTripRemovesExpenses(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	ana, _ := twoUsers(t, app)
	trip := createTrip(t, c, srv)
	do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/expenses", srv.URL, trip.ID), expense(ana, 100, "BRL"))
	do(t, c, "DELETE", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)
	var n int
	app.db.QueryRow(`SELECT count(*) FROM expenses`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d left", n)
	}
}
