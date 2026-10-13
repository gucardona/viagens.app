package main

import (
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Attachment struct {
	ID          int64  `json:"id"`
	TripID      int64  `json:"trip_id"`
	ItemID      *int64 `json:"item_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"created_at"`
}

const attCols = `id, trip_id, item_id, filename, content_type, size, created_at`

func (a *App) listAttachments(tripID int64) ([]Attachment, error) {
	rows, err := a.db.Query(`SELECT `+attCols+` FROM attachments WHERE trip_id = ? ORDER BY id`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	atts := []Attachment{}
	for rows.Next() {
		var at Attachment
		if err := rows.Scan(&at.ID, &at.TripID, &at.ItemID, &at.Filename, &at.ContentType, &at.Size, &at.CreatedAt); err != nil {
			return nil, err
		}
		atts = append(atts, at)
	}
	return atts, rows.Err()
}

const maxUpload = 25 << 20

func (a *App) filePath(id int64) string {
	return filepath.Join(a.dataDir, "files", strconv.FormatInt(id, 10))
}

// removeFiles deletes stored files; a failure is logged and does not stop the rest.
func (a *App) removeFiles(ids []int64) {
	for _, id := range ids {
		if err := os.Remove(a.filePath(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.Error("remove file", "id", id, "err", err)
		}
	}
}

func (a *App) attachmentIDs(query string, arg int64) ([]int64, error) {
	rows, err := a.db.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	tripID, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, ok := a.getTrip(w, tripID); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, 413, "Arquivo maior que 25 MB")
		} else {
			writeError(w, 400, "Envio inválido")
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "Escolha um arquivo")
		return
	}
	defer file.Close()
	if header.Size > maxUpload {
		writeError(w, 413, "Arquivo maior que 25 MB")
		return
	}
	var itemID *int64
	if s := r.FormValue("item_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		var owner int64
		if err == nil {
			err = a.db.QueryRow(`SELECT trip_id FROM items WHERE id = ?`, id).Scan(&owner)
		}
		if err != nil || owner != tripID {
			writeError(w, 400, "Item inválido")
			return
		}
		itemID = &id
	}
	ctype := header.Header.Get("Content-Type")
	if ctype == "" {
		head := make([]byte, 512)
		n, _ := io.ReadFull(file, head)
		ctype = http.DetectContentType(head[:n])
		file.Seek(0, io.SeekStart)
	}

	tmp, err := os.CreateTemp(filepath.Join(a.dataDir, "files"), ".upload-*")
	if err != nil {
		serverError(w, err)
		return
	}
	defer os.Remove(tmp.Name()) // no-op after the rename
	tmp.Chmod(0o644) // readable by the host user that runs the backup rsync
	size, err := io.Copy(tmp, file)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		serverError(w, err)
		return
	}
	name := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	res, err := a.db.Exec(`INSERT INTO attachments (trip_id, item_id, filename, content_type, size) VALUES (?, ?, ?, ?, ?)`,
		tripID, itemID, name, ctype, size)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	if err := os.Rename(tmp.Name(), a.filePath(id)); err != nil {
		a.db.Exec(`DELETE FROM attachments WHERE id = ?`, id)
		serverError(w, err)
		return
	}
	at, err := a.getAttachment(id)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, at)
}

func (a *App) getAttachment(id int64) (Attachment, error) {
	var at Attachment
	err := a.db.QueryRow(`SELECT `+attCols+` FROM attachments WHERE id = ?`, id).
		Scan(&at.ID, &at.TripID, &at.ItemID, &at.Filename, &at.ContentType, &at.Size, &at.CreatedAt)
	return at, err
}

// inlineTypes are shown in the browser; anything else (html, svg...) downloads, so an upload can't run scripts on our origin.
var inlineTypes = map[string]bool{"application/pdf": true, "image/jpeg": true, "image/png": true,
	"image/gif": true, "image/webp": true, "image/heic": true, "text/plain": true}

func (a *App) handleDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	at, err := a.getAttachment(id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "Arquivo não encontrado")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	f, err := os.Open(a.filePath(id))
	if err != nil {
		serverError(w, err)
		return
	}
	defer f.Close()
	disp := "attachment"
	if mt, _, _ := mime.ParseMediaType(at.ContentType); inlineTypes[mt] {
		disp = "inline"
	}
	w.Header().Set("Content-Type", at.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disp, map[string]string{"filename": at.Filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "", time.Time{}, f)
}

func (a *App) handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := a.db.Exec(`DELETE FROM attachments WHERE id = ?`, id); err != nil {
		serverError(w, err)
		return
	}
	a.removeFiles([]int64{id})
	w.WriteHeader(204)
}
