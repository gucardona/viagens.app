package main

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strings"
)

type Expense struct {
	ID          int64    `json:"id"`
	TripID      int64    `json:"trip_id"`
	ItemID      *int64   `json:"item_id"`
	Title       string   `json:"title"`
	AmountCents int64    `json:"amount_cents"`
	Currency    string   `json:"currency"`
	Category    string   `json:"category"`
	PaidBy      int64    `json:"paid_by"`
	PayerShare  int      `json:"payer_share"`
	Date        string   `json:"date"`
	Rate        *float64 `json:"rate"`
	CreatedAt   string   `json:"created_at"`
	BRLCents    *int64   `json:"brl_cents,omitempty"` // only in the trip summary; nil = no exchange rate
}

type expenseInput struct {
	ItemID      *int64   `json:"item_id"`
	Title       string   `json:"title"`
	AmountCents int64    `json:"amount_cents"`
	Currency    string   `json:"currency"`
	Category    string   `json:"category"`
	PaidBy      int64    `json:"paid_by"`
	PayerShare  *int     `json:"payer_share"`
	Date        string   `json:"date"`
	Rate        *float64 `json:"rate"`
}

var categories = map[string]bool{"lodging": true, "transport": true, "food": true, "activity": true, "shopping": true, "other": true}
var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

func (a *App) validateExpense(tripID int64, in *expenseInput) string {
	in.Title = strings.TrimSpace(in.Title)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.PayerShare == nil {
		half := 50
		in.PayerShare = &half
	}
	var n int
	switch {
	case in.Title == "" || len([]rune(in.Title)) > 200:
		return "Informe uma descrição de até 200 caracteres"
	case in.AmountCents <= 0 || in.AmountCents > 1e11:
		return "Valor inválido"
	case !currencyRe.MatchString(in.Currency):
		return "Moeda inválida"
	case !categories[in.Category]:
		return "Categoria inválida"
	case *in.PayerShare < 0 || *in.PayerShare > 100:
		return "Divisão inválida"
	case in.Rate != nil && *in.Rate <= 0:
		return "Cotação inválida"
	}
	if _, ok := parseDate(in.Date); !ok {
		return "Data inválida"
	}
	if a.db.QueryRow(`SELECT count(*) FROM users WHERE id = ?`, in.PaidBy).Scan(&n); n == 0 {
		return "Escolha quem pagou"
	}
	if in.ItemID != nil {
		if a.db.QueryRow(`SELECT count(*) FROM items WHERE id = ? AND trip_id = ?`, *in.ItemID, tripID).Scan(&n); n == 0 {
			return "Item inválido"
		}
	}
	return ""
}

const expenseCols = `id, trip_id, item_id, title, amount_cents, currency, category, paid_by, payer_share, date, rate, created_at`

func scanExpense(row interface{ Scan(...any) error }) (Expense, error) {
	var e Expense
	err := row.Scan(&e.ID, &e.TripID, &e.ItemID, &e.Title, &e.AmountCents, &e.Currency, &e.Category, &e.PaidBy, &e.PayerShare, &e.Date, &e.Rate, &e.CreatedAt)
	return e, err
}

func (a *App) getExpense(w http.ResponseWriter, id int64) (Expense, bool) {
	e, err := scanExpense(a.db.QueryRow(`SELECT `+expenseCols+` FROM expenses WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "Gasto não encontrado")
		return e, false
	}
	if err != nil {
		serverError(w, err)
		return e, false
	}
	return e, true
}

func (a *App) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	var in expenseInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := a.validateExpense(tripID, &in); msg != "" {
		writeError(w, 400, msg)
		return
	}
	res, err := a.db.Exec(`INSERT INTO expenses (trip_id, item_id, title, amount_cents, currency, category, paid_by, payer_share, date, rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tripID, in.ItemID, in.Title, in.AmountCents, in.Currency, in.Category, in.PaidBy, *in.PayerShare, in.Date, in.Rate)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	if e, ok := a.getExpense(w, id); ok {
		writeJSON(w, 201, e)
	}
}

func (a *App) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	old, ok := a.getExpense(w, id)
	if !ok {
		return
	}
	var in expenseInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := a.validateExpense(old.TripID, &in); msg != "" {
		writeError(w, 400, msg)
		return
	}
	if _, err := a.db.Exec(`UPDATE expenses SET item_id = ?, title = ?, amount_cents = ?, currency = ?, category = ?, paid_by = ?,
		payer_share = ?, date = ?, rate = ? WHERE id = ?`,
		in.ItemID, in.Title, in.AmountCents, in.Currency, in.Category, in.PaidBy, *in.PayerShare, in.Date, in.Rate, id); err != nil {
		serverError(w, err)
		return
	}
	if e, ok := a.getExpense(w, id); ok {
		writeJSON(w, 200, e)
	}
}

func (a *App) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := a.db.Exec(`DELETE FROM expenses WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`SELECT id, name FROM users ORDER BY id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	type user struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	out := []user{}
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, u)
	}
	writeJSON(w, 200, out)
}

type Balance struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Cents int64  `json:"cents"`
}

type ExpenseSummary struct {
	Expenses    []Expense        `json:"expenses"`
	TotalCents  int64            `json:"total_cents"`
	ByCategory  map[string]int64 `json:"by_category"`
	ByDay       map[string]int64 `json:"by_day"`
	MissingRate int              `json:"missing_rate"`
	Balance     *Balance         `json:"balance"`
}

// handleListExpenses returns the trip's expenses converted to BRL, totals and who owes whom.
func (a *App) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT `+expenseCols+` FROM expenses WHERE trip_id = ? ORDER BY date, id`, tripID)
	if err != nil {
		serverError(w, err)
		return
	}
	s := ExpenseSummary{Expenses: []Expense{}, ByCategory: map[string]int64{}, ByDay: map[string]int64{}}
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		s.Expenses = append(s.Expenses, e)
	}
	rows.Close()

	var names []string
	var ids []int64
	urows, err := a.db.Query(`SELECT id, name FROM users ORDER BY id`)
	if err != nil {
		serverError(w, err)
		return
	}
	for urows.Next() {
		var id int64
		var n string
		urows.Scan(&id, &n)
		ids, names = append(ids, id), append(names, n)
	}
	urows.Close()

	// shortcut: the balance assumes exactly two accounts (the couple); with more users it is omitted.
	owed := 0.0 // positive: ids[1] owes ids[0]
	for i := range s.Expenses {
		e := &s.Expenses[i]
		rate := 1.0
		if e.Rate != nil {
			rate = *e.Rate
		} else if e.Currency != "BRL" {
			if rate, err = a.rate(r.Context(), e.Date, e.Currency); err != nil {
				s.MissingRate++
				continue
			}
		}
		brl := float64(e.AmountCents) * rate
		cents := int64(math.Round(brl))
		e.BRLCents = &cents
		s.TotalCents += cents
		s.ByCategory[e.Category] += cents
		s.ByDay[e.Date] += cents
		other := brl * float64(100-e.PayerShare) / 100
		if len(ids) == 2 && e.PaidBy == ids[0] {
			owed += other
		} else if len(ids) == 2 && e.PaidBy == ids[1] {
			owed -= other
		}
	}
	if c := int64(math.Round(math.Abs(owed))); len(ids) == 2 && c > 0 {
		if owed > 0 {
			s.Balance = &Balance{From: names[1], To: names[0], Cents: c}
		} else {
			s.Balance = &Balance{From: names[0], To: names[1], Cents: c}
		}
	}
	writeJSON(w, 200, s)
}
