package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// trmnlNextRace is the upcoming race section of the next-race payload.
type trmnlNextRace struct {
	RaceDate      string `json:"race_date"`
	Track         string `json:"track,omitempty"`
	Country       string `json:"country,omitempty"`
	TrackID       string `json:"track_id,omitempty"`
	TotalLaps     int    `json:"total_laps,omitempty"`
	DaysRemaining int    `json:"days_remaining"`
}

// GetTRMNLNextRace returns a race-weekend payload for the TRMNL e-ink plugin:
// the configured upcoming race (with a server-side countdown), the latest
// finalized round (top 3), and the top 5 of the season championship
// standings. When no upcoming race is configured it still responds 200 with
// next_race: null so the display can show a "not scheduled" state.
func (h *Handler) GetTRMNLNextRace(c *gin.Context) {
	// Next race: the most recent race_info row's configured event.
	var nextRace *trmnlNextRace
	var country, track, trackID, nextRaceDate string
	var laps int
	err := h.S.DB.QueryRow(`
		SELECT COALESCE(country, ''), COALESCE(track, ''), COALESCE(track_id, ''),
			COALESCE(laps, 0), COALESCE(next_race_date, '')
		FROM race_info
		ORDER BY id DESC
		LIMIT 1`).Scan(&country, &track, &trackID, &laps, &nextRaceDate)
	switch {
	case err == sql.ErrNoRows:
		// No race_info row: next_race stays null.
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
		if nextRaceDate != "" {
			nextRace = &trmnlNextRace{
				RaceDate:      nextRaceDate,
				Track:         track,
				Country:       country,
				TrackID:       trackID,
				TotalLaps:     laps,
				DaysRemaining: trmnlDaysRemaining(nextRaceDate),
			}
		}
	}

	latestRace := h.trmnlLatestRace(c, 3)
	season, seasonID := h.trmnlSeason()

	standings := make([]models.SeasonStanding, 0)
	if season != nil {
		standings = h.trmnlStandings(c, seasonID, 5)
	}

	c.JSON(http.StatusOK, gin.H{
		"next_race":   nextRace,
		"latest_race": latestRace,
		"standings":   standings,
		"season":      season,
	})
}

// trmnlDaysRemaining computes the number of whole days until a YYYY-MM-DD
// race date, clamped at zero for today or past dates. An unset or unparsable
// date yields 0.
func trmnlDaysRemaining(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	days := int(math.Ceil(t.Sub(today).Hours() / 24))
	if days < 0 {
		return 0
	}
	return days
}
