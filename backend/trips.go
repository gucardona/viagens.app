package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Trip struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Destination string   `json:"destination"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	Notes       string   `json:"notes"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Countries   []string `json:"countries,omitempty"`
	ShareToken  string   `json:"share_token,omitempty"`
}

type TripDetail struct {
	Trip        Trip         `json:"trip"`
	Items       []Item       `json:"items"`
	Attachments []Attachment `json:"attachments"`
}

type tripInput struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Notes       string `json:"notes"`
}

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil
}

func (in *tripInput) validate() string {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 200 {
		return "Informe o nome da viagem"
	}
	start, ok1 := parseDate(in.StartDate)
	end, ok2 := parseDate(in.EndDate)
	if !ok1 || !ok2 {
		return "Datas inválidas"
	}
	if end.Before(start) {
		return "A data de fim é antes do início"
	}
	if end.Sub(start) > 365*24*time.Hour {
		return "A viagem pode ter no máximo um ano"
	}
	return ""
}

const tripCols = `id, name, destination, start_date, end_date, notes, created_at, updated_at, COALESCE(share_token, '')`

func scanTrip(row interface{ Scan(...any) error }) (Trip, error) {
	var t Trip
	err := row.Scan(&t.ID, &t.Name, &t.Destination, &t.StartDate, &t.EndDate, &t.Notes, &t.CreatedAt, &t.UpdatedAt, &t.ShareToken)
	return t, err
}

func (a *App) getTrip(w http.ResponseWriter, id int64) (Trip, bool) {
	t, err := scanTrip(a.db.QueryRow(`SELECT `+tripCols+` FROM trips WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "Viagem não encontrada")
		return t, false
	}
	if err != nil {
		serverError(w, err)
		return t, false
	}
	return t, true
}

func (a *App) handleListTrips(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`SELECT ` + tripCols + ` FROM trips ORDER BY start_date, id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	trips := []Trip{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			serverError(w, err)
			return
		}
		trips = append(trips, t)
	}
	countries, err := a.tripCountries()
	if err != nil {
		serverError(w, err)
		return
	}
	for i := range trips {
		trips[i].Countries = countries[trips[i].ID]
	}
	writeJSON(w, 200, trips)
}

func (a *App) handleGetTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	trip, ok := a.getTrip(w, id)
	if !ok {
		return
	}
	items, err := a.listItems(id)
	if err != nil {
		serverError(w, err)
		return
	}
	atts, err := a.listAttachments(id)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, TripDetail{Trip: trip, Items: items, Attachments: atts})
}

func (a *App) handleCreateTrip(w http.ResponseWriter, r *http.Request) {
	var in tripInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, 400, msg)
		return
	}
	res, err := a.db.Exec(`INSERT INTO trips (name, destination, start_date, end_date, notes) VALUES (?, ?, ?, ?, ?)`,
		in.Name, in.Destination, in.StartDate, in.EndDate, in.Notes)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	if t, ok := a.getTrip(w, id); ok {
		writeJSON(w, 201, t)
	}
}

func (a *App) handleUpdateTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in tripInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeError(w, 400, msg)
		return
	}
	if _, err := a.db.Exec(`UPDATE trips SET name = ?, destination = ?, start_date = ?, end_date = ?, notes = ?, updated_at = datetime('now') WHERE id = ?`,
		in.Name, in.Destination, in.StartDate, in.EndDate, in.Notes, id); err != nil {
		serverError(w, err)
		return
	}
	if t, ok := a.getTrip(w, id); ok {
		writeJSON(w, 200, t)
	}
}

func (a *App) handleDeleteTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	files, err := a.attachmentIDs(`SELECT id FROM attachments WHERE trip_id = ?`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if _, err := a.db.Exec(`DELETE FROM trips WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	a.removeFiles(files)
	w.WriteHeader(204)
}

// tripCountries maps trip id to the sorted country codes of its items.
func (a *App) tripCountries() (map[int64][]string, error) {
	rows, err := a.db.Query(`SELECT DISTINCT trip_id, country_code FROM items WHERE country_code != '' ORDER BY trip_id, country_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var id int64
		var c string
		if err := rows.Scan(&id, &c); err != nil {
			return nil, err
		}
		out[id] = append(out[id], c)
	}
	return out, rows.Err()
}
