package main

import (
	"net/http"
	"strings"
)

type CheckItem struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

func (a *App) checklist(w http.ResponseWriter, tripID int64) {
	rows, err := a.db.Query(`SELECT id, text, done FROM checklist WHERE trip_id = ? ORDER BY id`, tripID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []CheckItem{}
	for rows.Next() {
		var c CheckItem
		if err := rows.Scan(&c.ID, &c.Text, &c.Done); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, c)
	}
	writeJSON(w, 200, out)
}

func (a *App) handleListChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, id); ok {
		a.checklist(w, id)
	}
}

func checkText(w http.ResponseWriter, s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len([]rune(s)) > 200 {
		writeError(w, 400, "Escreva um item de até 200 caracteres")
		return "", false
	}
	return s, true
}

func (a *App) handleAddChecklist(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	var in struct{ Text string }
	if !readJSON(w, r, &in) {
		return
	}
	text, ok := checkText(w, in.Text)
	if !ok {
		return
	}
	res, err := a.db.Exec(`INSERT INTO checklist (trip_id, text) VALUES (?, ?)`, tripID, text)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 201, CheckItem{ID: id, Text: text})
}

func (a *App) handleUpdateChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
		Done bool   `json:"done"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	text, ok := checkText(w, in.Text)
	if !ok {
		return
	}
	res, err := a.db.Exec(`UPDATE checklist SET text = ?, done = ? WHERE id = ?`, text, in.Done, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, 404, "Item não encontrado")
		return
	}
	writeJSON(w, 200, CheckItem{ID: id, Text: text, Done: in.Done})
}

func (a *App) handleDeleteChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := a.db.Exec(`DELETE FROM checklist WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

// handleCopyChecklist adds another trip's texts, unchecked, skipping ones this trip already has.
func (a *App) handleCopyChecklist(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	var in struct {
		FromTripID int64 `json:"from_trip_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if _, err := a.db.Exec(`INSERT INTO checklist (trip_id, text)
		SELECT ?, src.text FROM checklist src WHERE src.trip_id = ?
		AND lower(src.text) NOT IN (SELECT lower(text) FROM checklist WHERE trip_id = ?) ORDER BY src.id`,
		tripID, in.FromTripID, tripID); err != nil {
		serverError(w, err)
		return
	}
	a.checklist(w, tripID)
}
