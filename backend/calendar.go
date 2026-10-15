package main

import (
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// handleCalendar exports the dated items of a trip as an iCalendar file (RFC 5545).
// shortcut: floating times (no TZID), so the phone shows them in its own zone, which is the
// destination's during the trip but home's before it; store a zone per item if that matters.
func (a *App) handleCalendar(w http.ResponseWriter, r *http.Request) {
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
	var b strings.Builder
	line := func(s string) { b.WriteString(fold(s) + "\r\n") }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//viagens.app//PT-BR")
	line("CALSCALE:GREGORIAN")
	line("X-WR-CALNAME:" + icsText(trip.Name))
	for _, i := range items {
		start, ok := parseDate(i.StartDate)
		if !ok {
			continue
		}
		line("BEGIN:VEVENT")
		line(fmt.Sprintf("UID:item-%d@viagens.app", i.ID))
		if t, err := time.Parse(time.DateTime, i.UpdatedAt); err == nil {
			line("DTSTAMP:" + t.UTC().Format("20060102T150405Z"))
		}
		line("SUMMARY:" + icsText(i.Title))
		if i.StartTime == "" {
			end := start
			if e, ok := parseDate(i.EndDate); ok {
				end = e
			}
			line("DTSTART;VALUE=DATE:" + start.Format("20060102"))
			line("DTEND;VALUE=DATE:" + end.AddDate(0, 0, 1).Format("20060102"))
		} else {
			from := start.Format("20060102") + "T" + strings.ReplaceAll(i.StartTime, ":", "") + "00"
			line("DTSTART:" + from)
			to := ""
			if i.EndTime != "" {
				d := i.EndDate
				if d == "" {
					d = i.StartDate
				}
				to = strings.ReplaceAll(d, "-", "") + "T" + strings.ReplaceAll(i.EndTime, ":", "") + "00"
			}
			if to > from {
				line("DTEND:" + to)
			} else {
				line("DURATION:PT1H")
			}
		}
		if i.Address != "" {
			line("LOCATION:" + icsText(i.Address))
		}
		if i.Lat != nil && i.Lng != nil {
			line(fmt.Sprintf("GEO:%g;%g", *i.Lat, *i.Lng))
		}
		var desc []string
		if i.ConfirmationCode != "" {
			desc = append(desc, "Confirmação: "+i.ConfirmationCode)
		}
		switch i.Ticket {
		case "needed":
			desc = append(desc, "Comprar ingresso")
		case "bought":
			desc = append(desc, "Ingresso comprado")
		}
		if i.TimeApprox {
			desc = append(desc, "Horário aproximado")
		}
		if i.Notes != "" {
			desc = append(desc, i.Notes)
		}
		if len(desc) > 0 {
			line("DESCRIPTION:" + icsText(strings.Join(desc, "\n")))
		}
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": trip.Name + ".ics"}))
	w.Write([]byte(b.String()))
}

var icsEscaper = strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`)

func icsText(s string) string { return icsEscaper.Replace(s) }

// fold splits a content line into lines of at most 75 octets, never inside a UTF-8 character.
func fold(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		size := utf8.RuneLen(r)
		if n+size > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}
