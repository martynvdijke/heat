package racing

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ICALEvent is an all-day event keyed on Date.
type ICALEvent struct {
	UID         string
	Date        time.Time
	Summary     string
	Description string
	Location    string
}

// BuildRacesICS builds an iCalendar string per RFC 5545.
func BuildRacesICS(events []ICALEvent, calendarName string, now time.Time) string {
	// Sort chronologically ascending.
	cp := make([]ICALEvent, len(events))
	copy(cp, events)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Date.Before(cp[j].Date)
	})

	var sb strings.Builder
	w := func(s string) {
		sb.WriteString(foldLine(s))
		sb.WriteString("\r\n")
	}
	w("BEGIN:VCALENDAR")
	w("VERSION:2.0")
	w("PRODID:-//HEAT//Racer Calendar//EN")
	w("CALSCALE:GREGORIAN")
	w("METHOD:PUBLISH")
	w("X-WR-CALNAME:" + escapeText(calendarName))

	dtStamp := now.UTC().Format("20060102T150405Z")

	for _, ev := range cp {
		w("BEGIN:VEVENT")
		w("UID:" + escapeText(ev.UID))
		w("DTSTAMP:" + dtStamp)
		w("DTSTART;VALUE=DATE:" + ev.Date.Format("20060102"))
		w("DTEND;VALUE=DATE:" + ev.Date.AddDate(0, 0, 1).Format("20060102"))
		w("SUMMARY:" + escapeText(ev.Summary))
		if ev.Description != "" {
			w("DESCRIPTION:" + escapeText(ev.Description))
		}
		if ev.Location != "" {
			w("LOCATION:" + escapeText(ev.Location))
		}
		w("END:VEVENT")
	}
	w("END:VCALENDAR")
	return sb.String()
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// foldLine folds a single logical line per RFC 5545: >75 octets => CRLF + SP.
func foldLine(s string) string {
	if len(s) <= 75 {
		return s
	}
	var out strings.Builder
	// Fold by bytes, avoid splitting multi-byte rune.
	pos := 0
	first := true
	for pos < len(s) {
		limit := 75
		if !first {
			// continuation lines have leading space counted? spec: 75 octets per line including content; folding inserts CRLF+SP, next line starts with SP then up to 74 octets of content. Simpler: fold at 75 bytes for first, 74 for continuation to keep total line <=75? But task says fold lines longer than 75 octets using CRLF+SP on continuation. We'll fold at 75 for first, 74 for continuation content.
			limit = 74
		}
		if pos+limit >= len(s) {
			out.WriteString(s[pos:])
			break
		}
		end := pos + limit
		// Avoid splitting rune: back up to rune boundary.
		for end > pos && !utf8.RuneStart(s[end]) {
			end--
		}
		// If we backed up too far (e.g., limit inside a 4-byte rune, end==pos), just cut at limit (will split rune? fallback to limit)
		if end == pos {
			end = pos + limit
		}
		out.WriteString(s[pos:end])
		out.WriteString("\r\n ")
		pos = end
		first = false
	}
	return out.String()
}
