package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// hub tells every open app that something changed, so it refetches what it shows.
// shortcut: in-memory, so it only works with one server process; move to the database if that changes.
type hub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func (h *hub) subscribe() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() { h.mu.Lock(); delete(h.subs, ch); h.mu.Unlock() }
}

func (h *hub) publish() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default: // a change is already pending for this one
		}
	}
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	flush := w.(http.Flusher).Flush
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, done := a.events.subscribe()
	defer done()
	fmt.Fprint(w, ": ok\n\n")
	flush()
	ping := time.NewTicker(25 * time.Second) // keeps proxies from closing an idle stream
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprint(w, "data: change\n\n")
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		flush()
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) { s.status = code; s.ResponseWriter.WriteHeader(code) }

// notifyChanges publishes after every successful API write.
func (a *App) notifyChanges(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		sw := &statusWriter{w, 200}
		next.ServeHTTP(sw, r)
		if sw.status < 400 {
			a.events.publish()
		}
	})
}
