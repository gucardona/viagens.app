package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// listen opens /api/events and returns a channel with each "data:" line received.
func listen(t *testing.T, c *http.Client, url string) <-chan string {
	t.Helper()
	req, _ := http.NewRequest("GET", url+"/api/events", nil)
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("events: %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	got := make(chan string, 10)
	go func() {
		s := bufio.NewScanner(res.Body)
		for s.Scan() {
			if strings.HasPrefix(s.Text(), "data:") {
				got <- s.Text()
			}
		}
	}()
	return got
}

func expectEvent(t *testing.T, ch <-chan string, want bool) {
	t.Helper()
	select {
	case line := <-ch:
		if !want {
			t.Fatalf("unexpected event %q", line)
		}
		if line != "data: change" {
			t.Fatalf("event %q", line)
		}
	case <-time.After(300 * time.Millisecond):
		if want {
			t.Fatal("no event")
		}
	}
}

func TestEventsOnWrite(t *testing.T) {
	app, srv := newTestApp(t)
	c := login(t, app, srv)
	trip := createTrip(t, c, srv)
	a, b := listen(t, c, srv.URL), listen(t, c, srv.URL)
	time.Sleep(50 * time.Millisecond)

	do(t, c, "GET", srv.URL+"/api/trips", nil)
	expectEvent(t, a, false)

	if res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), hotel()); res.StatusCode != 201 {
		t.Fatalf("create item: %d", res.StatusCode)
	}
	expectEvent(t, a, true)
	expectEvent(t, b, true)

	if res := do(t, c, "POST", fmt.Sprintf("%s/api/trips/%d/items", srv.URL, trip.ID), map[string]any{"kind": "x"}); res.StatusCode != 400 {
		t.Fatalf("bad item: %d", res.StatusCode)
	}
	expectEvent(t, a, false)
}

func TestEventsRequireSession(t *testing.T) {
	_, srv := newTestApp(t)
	if res := do(t, newClient(), "GET", srv.URL+"/api/events", nil); res.StatusCode != 401 {
		t.Fatalf("status %d", res.StatusCode)
	}
}
