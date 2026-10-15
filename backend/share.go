package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
)

// SharedTrip is what the read-only family link shows: the dated itinerary,
// without notes, confirmation codes, documents or expenses.
type SharedTrip struct {
	Trip  Trip   `json:"trip"`
	Items []Item `json:"items"`
}

func (a *App) handleShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	trip, ok := a.getTrip(w, id)
	if !ok {
		return
	}
	if trip.ShareToken == "" {
		b := make([]byte, 24)
		rand.Read(b)
		trip.ShareToken = base64.RawURLEncoding.EncodeToString(b)
		if _, err := a.db.Exec(`UPDATE trips SET share_token = ? WHERE id = ?`, trip.ShareToken, id); err != nil {
			serverError(w, err)
			return
		}
	}
	writeJSON(w, 200, map[string]string{"share_token": trip.ShareToken})
}

func (a *App) handleUnshare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := a.db.Exec(`UPDATE trips SET share_token = NULL WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (a *App) handleShared(w http.ResponseWriter, r *http.Request) {
	trip, err := scanTrip(a.db.QueryRow(`SELECT `+tripCols+` FROM trips WHERE share_token = ?`, r.PathValue("token")))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "Link inválido ou desativado")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	items, err := a.listItems(trip.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	trip.Notes, trip.ShareToken = "", ""
	out := SharedTrip{Trip: trip, Items: []Item{}}
	for _, i := range items {
		if i.StartDate != "" {
			i.ConfirmationCode, i.Notes = "", ""
			out.Items = append(out.Items, i)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
