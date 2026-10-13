package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func upload(t *testing.T, c *http.Client, url, filename, contentType string, data []byte, itemID int64) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if itemID != 0 {
		mw.WriteField("item_id", fmt.Sprint(itemID))
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename}))
	h.Set("Content-Type", contentType)
	part, _ := mw.CreatePart(h)
	part.Write(data)
	mw.Close()
	req, _ := http.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Requested-With", "viagens")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func countFiles(t *testing.T, app *App) int {
	entries, err := os.ReadDir(filepath.Join(app.dataDir, "files"))
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestUploadAndDownload(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	pdf := []byte("%PDF-1.4 teste")

	res := upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "Passagem São Paulo.pdf", "application/pdf", pdf, 0)
	if res.StatusCode != 201 {
		t.Fatalf("upload: %d", res.StatusCode)
	}
	att := decode[Attachment](t, res)
	if att.Size != int64(len(pdf)) || att.ItemID != nil {
		t.Fatalf("att: %+v", att)
	}

	res = do(t, c, "GET", fmt.Sprintf("%s/api/attachments/%d", srv.URL, att.ID), nil)
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !bytes.Equal(body, pdf) || res.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("download: %d %q %s", res.StatusCode, body, res.Header.Get("Content-Type"))
	}
	_, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if err != nil || params["filename"] != "Passagem São Paulo.pdf" {
		t.Fatalf("disposition: %q %v", res.Header.Get("Content-Disposition"), err)
	}

	detail := decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil))
	if len(detail.Attachments) != 1 {
		t.Fatalf("detail attachments: %+v", detail.Attachments)
	}

	if res := do(t, c, "DELETE", fmt.Sprintf("%s/api/attachments/%d", srv.URL, att.ID), nil); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	if n := countFiles(t, app); n != 0 {
		t.Fatalf("files left: %d", n)
	}
}

func TestUploadPathTraversal(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	res := upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "../../evil.txt", "text/plain", []byte("x"), 0)
	if res.StatusCode != 201 {
		t.Fatalf("upload: %d", res.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(app.dataDir, "..", "evil.txt")); err == nil {
		t.Fatal("file written outside data dir")
	}
	if _, err := os.Stat(filepath.Join(app.dataDir, "evil.txt")); err == nil {
		t.Fatal("file written with client name")
	}
	if n := countFiles(t, app); n != 1 {
		t.Fatalf("files: %d", n)
	}
}

func TestUploadTooLarge(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	big := bytes.Repeat([]byte("a"), 25<<20+1)
	res := upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "big.bin", "application/octet-stream", big, 0)
	if res.StatusCode != 413 {
		t.Fatalf("status %d", res.StatusCode)
	}
	var n int
	app.db.QueryRow(`SELECT count(*) FROM attachments`).Scan(&n)
	if n != 0 || countFiles(t, app) != 0 {
		t.Fatalf("rows %d files %d", n, countFiles(t, app))
	}
}

func TestDownloadNeedsSession(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	att := decode[Attachment](t, upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "a.pdf", "application/pdf", []byte("x"), 0))
	if res := do(t, newClient(), "GET", fmt.Sprintf("%s/api/attachments/%d", srv.URL, att.ID), nil); res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestUploadItemFromOtherTrip(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip1 := createTrip(t, c, srv)
	trip2 := createTrip(t, c, srv)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip2.ID), hotel()))
	res := upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip1.ID), "a.pdf", "application/pdf", []byte("x"), item.ID)
	if res.StatusCode != 400 {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestDeleteTripRemovesFiles(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))
	url := fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID)
	upload(t, c, url, "geral.pdf", "application/pdf", []byte("x"), 0)
	upload(t, c, url, "item.pdf", "application/pdf", []byte("y"), item.ID)
	if n := countFiles(t, app); n != 2 {
		t.Fatalf("files before: %d", n)
	}
	do(t, c, "DELETE", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)
	if n := countFiles(t, app); n != 0 {
		t.Fatalf("files after: %d", n)
	}
}

func TestDeleteItemRemovesItsFiles(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	item := decode[Item](t, do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()))
	url := fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID)
	upload(t, c, url, "geral.pdf", "application/pdf", []byte("x"), 0)
	upload(t, c, url, "item.pdf", "application/pdf", []byte("y"), item.ID)
	do(t, c, "DELETE", fmt.Sprintf("%s/api/items/%d", srv.URL, item.ID), nil)
	if n := countFiles(t, app); n != 1 {
		t.Fatalf("files after: %d", n)
	}
	if !strings.Contains(fmt.Sprint(decode[TripDetail](t, do(t, c, "GET", fmt.Sprintf("%s/api/trips/%d", srv.URL, trip.ID), nil)).Attachments), "geral.pdf") {
		t.Fatal("general attachment lost")
	}
}

func TestHTMLUploadIsNotRenderedInline(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	att := decode[Attachment](t, upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "x.html", "text/html", []byte("<script>alert(1)</script>"), 0))
	res := do(t, c, "GET", fmt.Sprintf("%s/api/attachments/%d", srv.URL, att.ID), nil)
	if d := res.Header.Get("Content-Disposition"); !strings.HasPrefix(d, "attachment") || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers: %q", d)
	}
}

func TestUploadedFileIsReadableByBackupUser(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	att := decode[Attachment](t, upload(t, c, fmt.Sprintf("%s/api/trips/%d/attachments", srv.URL, trip.ID), "a.pdf", "application/pdf", []byte("x"), 0))
	st, err := os.Stat(app.filePath(att.ID))
	if err != nil || st.Mode().Perm() != 0o644 {
		t.Fatalf("mode %v err %v", st.Mode().Perm(), err)
	}
}
