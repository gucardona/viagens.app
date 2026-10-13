package main

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	placeCoords = regexp.MustCompile(`!3d(-?\d+(?:\.\d+)?)!4d(-?\d+(?:\.\d+)?)`)
	atCoords    = regexp.MustCompile(`@(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)`)
	pairCoords  = regexp.MustCompile(`^(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)$`)
)

var googleHosts = map[string]bool{"google.com": true, "www.google.com": true, "maps.google.com": true, "consent.google.com": true}
var shortHosts = map[string]bool{"maps.app.goo.gl": true, "goo.gl": true}

func floats(m []string) (*float64, *float64) {
	lat, err1 := strconv.ParseFloat(m[1], 64)
	lng, err2 := strconv.ParseFloat(m[2], 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, nil
	}
	return &lat, &lng
}

// parseMapsURL reads a place name and coordinates from a full Google Maps link, without network.
func parseMapsURL(u *url.URL) (title string, lat, lng *float64) {
	if rest, ok := strings.CutPrefix(u.Path, "/maps/place/"); ok {
		title, _, _ = strings.Cut(rest, "/")
		title = strings.ReplaceAll(title, "+", " ")
	}
	q := u.Query().Get("q")
	if q == "" {
		q = u.Query().Get("query")
	}
	full := u.Path + "?" + u.RawQuery
	if m := placeCoords.FindStringSubmatch(full); m != nil {
		lat, lng = floats(m)
	} else if m := pairCoords.FindStringSubmatch(q); m != nil {
		lat, lng = floats(m)
	} else if m := atCoords.FindStringSubmatch(full); m != nil {
		lat, lng = floats(m)
	}
	if title == "" && q != "" && pairCoords.FindStringSubmatch(q) == nil {
		title = q
	}
	return strings.TrimSpace(title), lat, lng
}

var errBadLink = errors.New("bad link")

// resolveMapsLink follows short links only through Google hosts and stops at the first full Google URL.
func (a *App) resolveMapsLink(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errBadLink
	}
	if googleHosts[u.Host] {
		return unwrapConsent(u)
	}
	if !shortHosts[u.Host] {
		return nil, errBadLink
	}
	var final *url.URL
	client := *a.linkClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		switch {
		case len(via) > 5:
			return errBadLink
		case googleHosts[req.URL.Host]:
			final = req.URL
			return http.ErrUseLastResponse
		case shortHosts[req.URL.Host]:
			return nil
		}
		return errBadLink
	}
	res, err := client.Get(u.String())
	if err != nil {
		return nil, errBadLink
	}
	res.Body.Close()
	if final == nil {
		return nil, errBadLink
	}
	return unwrapConsent(final)
}

// unwrapConsent returns the real Maps URL behind Google's EU cookie-consent page.
func unwrapConsent(u *url.URL) (*url.URL, error) {
	if u.Host != "consent.google.com" {
		return u, nil
	}
	next, err := url.Parse(u.Query().Get("continue"))
	if err != nil || !googleHosts[next.Host] || next.Host == "consent.google.com" {
		return nil, errBadLink
	}
	return next, nil
}

func (a *App) handleMapsLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	u, err := a.resolveMapsLink(in.URL)
	if err != nil {
		writeError(w, 400, "Link do Google Maps inválido")
		return
	}
	title, lat, lng := parseMapsURL(u)
	writeJSON(w, 200, map[string]any{"title": title, "lat": lat, "lng": lng})
}
