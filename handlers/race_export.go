package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/racing"
)

// @Summary Export races as iCalendar
// @Description Export upcoming and done races as an iCalendar (.ics) feed
// @Tags Race
// @Produce text/calendar
// @Success 200 {string} string "iCalendar data"
// @Router /api/races/export.ics [get]
func (h *Handler) ExportRacesICS(c *gin.Context) {
	var events []racing.ICALEvent

	// Upcoming race
	var nextRaceDate, country, track string
	var laps int
	err := h.S.DB.QueryRow("SELECT COALESCE(next_race_date,''), country, track, laps FROM race_info ORDER BY id DESC LIMIT 1").Scan(&nextRaceDate, &country, &track, &laps)
	if err == nil && nextRaceDate != "" {
		if d, perr := time.Parse("2006-01-02", nextRaceDate); perr == nil {
			summaryTrack := track
			if summaryTrack == "" {
				summaryTrack = country
			}
			summary := fmt.Sprintf("HEAT Race: %s", summaryTrack)

			var descParts []string
			if track != "" && country != "" {
				descParts = append(descParts, fmt.Sprintf("Upcoming HEAT race at %s, %s", track, country))
			} else if track != "" {
				descParts = append(descParts, fmt.Sprintf("Upcoming HEAT race at %s", track))
			} else if country != "" {
				descParts = append(descParts, fmt.Sprintf("Upcoming HEAT race at %s", country))
			}
			if laps > 0 {
				if len(descParts) > 0 {
					descParts[0] = descParts[0] + fmt.Sprintf(" \u2014 %d laps.", laps)
				} else {
					descParts = append(descParts, fmt.Sprintf("%d laps.", laps))
				}
			} else {
				if len(descParts) > 0 && !strings.HasSuffix(descParts[0], ".") {
					descParts[0] = descParts[0] + "."
				}
			}
			description := strings.Join(descParts, " ")

			var loc string
			if track != "" && country != "" {
				loc = track + ", " + country
			} else if track != "" {
				loc = track
			} else {
				loc = country
			}
			loc = strings.Trim(loc, ", ")

			events = append(events, racing.ICALEvent{
				UID:         "next-race@heat-racer",
				Date:        d,
				Summary:     summary,
				Description: description,
				Location:    loc,
			})
		}
	} else if err != nil && err.Error() != "sql: no rows in result set" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Done races
	rows, err := h.S.DB.Query("SELECT id, COALESCE(name,''), race_date, country, track, total_laps, COALESCE(race_type,'season') FROM race_history ORDER BY race_date ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, totalLaps int
		var name, raceDate, rCountry, rTrack, raceType string
		if err := rows.Scan(&id, &name, &raceDate, &rCountry, &rTrack, &totalLaps, &raceType); err != nil {
			continue
		}
		d, perr := time.Parse("2006-01-02", raceDate)
		if perr != nil {
			continue
		}
		summary := name
		if summary == "" {
			trackPart := rTrack
			if trackPart == "" {
				trackPart = rCountry
			}
			summary = fmt.Sprintf("HEAT Race: %s", trackPart)
		}

		var descParts []string
		if rTrack != "" {
			descParts = append(descParts, rTrack)
		}
		if rCountry != "" {
			descParts = append(descParts, rCountry)
		}
		if totalLaps > 0 {
			descParts = append(descParts, fmt.Sprintf("%d laps", totalLaps))
		}
		if raceType != "" {
			descParts = append(descParts, raceType)
		}
		description := strings.Join(descParts, ", ")
		if description != "" {
			description = description + "."
		}

		var loc string
		if rTrack != "" && rCountry != "" {
			loc = rTrack + ", " + rCountry
		} else if rTrack != "" {
			loc = rTrack
		} else {
			loc = rCountry
		}
		loc = strings.Trim(loc, ", ")

		events = append(events, racing.ICALEvent{
			UID:         fmt.Sprintf("race-%d@heat-racer", id),
			Date:        d,
			Summary:     summary,
			Description: description,
			Location:    loc,
		})
	}

	ics := racing.BuildRacesICS(events, "HEAT Races", time.Now())
	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=heat_races.ics")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, ics)
}
