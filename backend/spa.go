package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed all:web/dist
var webFS embed.FS

// spaHandler serves built frontend files and falls back to index.html for client-side routes.
func spaHandler(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if st, err := fs.Stat(fsys, name); name == "" || err != nil || st.IsDir() {
			http.ServeFileFS(w, r, fsys, "index.html")
			return
		}
		files.ServeHTTP(w, r)
	})
}

// backup writes a consistent copy of the database to dest, replacing any previous copy.
func backup(db *sql.DB, dest string) error {
	tmp := dest + ".tmp"
	os.Remove(tmp)
	if _, err := db.Exec(`VACUUM INTO ?`, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
