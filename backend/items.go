package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Item struct {
	ID               int64    `json:"id"`
	TripID           int64    `json:"trip_id"`
	Kind             string   `json:"kind"`
	Title            string   `json:"title"`
	StartDate        string   `json:"start_date"`
	StartTime        string   `json:"start_time"`
	EndDate          string   `json:"end_date"`
	EndTime          string   `json:"end_time"`
	Address          string   `json:"address"`
	Lat              *float64 `json:"lat"`
	Lng              *float64 `json:"lng"`
	ConfirmationCode string   `json:"confirmation_code"`
	Notes            string   `json:"notes"`
	Ticket           string   `json:"ticket"`
	TimeApprox       bool     `json:"time_approx"`
	CountryCode      string   `json:"country_code"`
	Area             string   `json:"area"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type itemInput struct {
	Kind             string   `json:"kind"`
	Title            string   `json:"title"`
	StartDate        string   `json:"start_date"`
	StartTime        string   `json:"start_time"`
	EndDate          string   `json:"end_date"`
	EndTime          string   `json:"end_time"`
	Address          string   `json:"address"`
	Lat              *float64 `json:"lat"`
	Lng              *float64 `json:"lng"`
	ConfirmationCode string   `json:"confirmation_code"`
	Notes            string   `json:"notes"`
	Ticket           string   `json:"ticket"`
	TimeApprox       bool     `json:"time_approx"`
}

var tickets = map[string]bool{"": true, "needed": true, "bought": true}

var kinds = map[string]bool{"flight": true, "lodging": true, "car": true, "transport": true,
	"restaurant": true, "activity": true, "event": true, "note": true}

func validTime(s string) bool {
	_, err := time.Parse("15:04", s)
	return s == "" || (err == nil && len(s) == 5)
}

func (in *itemInput) validate() string {
	in.Title = strings.TrimSpace(in.Title)
	if !kinds[in.Kind] {
		return "Tipo inválido"
	}
	if !tickets[in.Ticket] {
		return "Ingresso inválido"
	}
	if in.Title == "" || len([]rune(in.Title)) > 200 {
		return "Informe um título de até 200 caracteres"
	}
	for _, d := range []string{in.StartDate, in.EndDate} {
		if _, ok := parseDate(d); d != "" && !ok {
			return "Data inválida"
		}
	}
	if !validTime(in.StartTime) || !validTime(in.EndTime) {
		return "Hora inválida"
	}
	if (in.StartTime != "" && in.StartDate == "") || (in.EndTime != "" && in.EndDate == "") {
		return "Informe a data junto com a hora"
	}
	if in.EndDate != "" && in.StartDate != "" && in.EndDate < in.StartDate {
		return "A data de fim é antes do início"
	}
	if (in.Lat == nil) != (in.Lng == nil) {
		return "Localização incompleta"
	}
	if in.Lat != nil && (*in.Lat < -90 || *in.Lat > 90 || *in.Lng < -180 || *in.Lng > 180) {
		return "Localização inválida"
	}
	return ""
}

const itemCols = `id, trip_id, kind, title, start_date, start_time, end_date, end_time, address, lat, lng, confirmation_code, notes, ticket, time_approx, country_code, area, created_at, updated_at`

func scanItem(row interface{ Scan(...any) error }) (Item, error) {
	var i Item
	err := row.Scan(&i.ID, &i.TripID, &i.Kind, &i.Title, &i.StartDate, &i.StartTime, &i.EndDate, &i.EndTime,
		&i.Address, &i.Lat, &i.Lng, &i.ConfirmationCode, &i.Notes, &i.Ticket, &i.TimeApprox, &i.CountryCode, &i.Area, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

func (a *App) listItems(tripID int64) ([]Item, error) {
	rows, err := a.db.Query(`SELECT `+itemCols+` FROM items WHERE trip_id = ? ORDER BY id`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		i, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (a *App) getItem(w http.ResponseWriter, id int64) (Item, bool) {
	i, err := scanItem(a.db.QueryRow(`SELECT `+itemCols+` FROM items WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "Item não encontrado")
		return i, false
	}
	if err != nil {
		serverError(w, err)
		return i, false
	}
	return i, true
}

func (a *App) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	var in itemInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, 400, msg)
		return
	}
	res, err := a.db.Exec(`INSERT INTO items (trip_id, kind, title, start_date, start_time, end_date, end_time, address, lat, lng, confirmation_code, notes, ticket, time_approx)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tripID, in.Kind, in.Title, in.StartDate, in.StartTime, in.EndDate, in.EndTime, in.Address, in.Lat, in.Lng, in.ConfirmationCode, in.Notes, in.Ticket, in.TimeApprox)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	a.kickGeo()
	if i, ok := a.getItem(w, id); ok {
		writeJSON(w, 201, i)
	}
}

func (a *App) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getItem(w, id); !ok {
		return
	}
	var in itemInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, 400, msg)
		return
	}
	if _, err := a.db.Exec(`UPDATE items SET kind = ?, title = ?, start_date = ?, start_time = ?, end_date = ?, end_time = ?,
		address = ?, lat = ?, lng = ?, confirmation_code = ?, notes = ?, ticket = ?, time_approx = ?, updated_at = datetime('now'),
		country_code = CASE WHEN lat IS ? AND lng IS ? THEN country_code ELSE '' END,
		area = CASE WHEN lat IS ? AND lng IS ? THEN area ELSE '' END,
		geo_key = CASE WHEN lat IS ? AND lng IS ? THEN geo_key ELSE '' END WHERE id = ?`,
		in.Kind, in.Title, in.StartDate, in.StartTime, in.EndDate, in.EndTime, in.Address, in.Lat, in.Lng, in.ConfirmationCode, in.Notes, in.Ticket, in.TimeApprox,
		in.Lat, in.Lng, in.Lat, in.Lng, in.Lat, in.Lng, id); err != nil {
		serverError(w, err)
		return
	}
	a.kickGeo()
	if i, ok := a.getItem(w, id); ok {
		writeJSON(w, 200, i)
	}
}

func (a *App) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	files, err := a.attachmentIDs(`SELECT id FROM attachments WHERE item_id = ?`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err := a.db.Exec(`DELETE FROM items WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	a.removeFiles(files)
	w.WriteHeader(204)
}
