package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"heat/models"
	"heat/racing"
)

// trmnlResult is a single finishing position inside the latest race payload.
type trmnlResult struct {
	RacerName      string `json:"racer_name"`
	Team           string `json:"team"`
	Position       int    `json:"position"`
	Points         int    `json:"points"`
	ProfilePicture string `json:"profile_picture,omitempty"`
}

// trmnlRace is the latest race section of the TRMNL summary payload.
type trmnlRace struct {
	Name      string        `json:"name"`
	RaceDate  string        `json:"race_date"`
	Round     int           `json:"round"`
	Country   string        `json:"country,omitempty"`
	Track     string        `json:"track,omitempty"`
	TotalLaps int           `json:"total_laps,omitempty"`
	Results   []trmnlResult `json:"results"`
}

// trmnlSeason is the season metadata section of the TRMNL summary payload.
type trmnlSeason struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GetTRMNLSummary returns a compact payload for TRMNL e-ink device polling:
// the latest finalized round (finishing order with points) and the current
// season championship standings. When no finalized rounds or no seasons
// exist, it still responds 200 with latest_race: null and an empty
// standings array.
func (h *Handler) GetTRMNLSummary(c *gin.Context) {
	// Latest race: most recent finalized round snapshot, tie-broken by round.
	var latestRace *trmnlRace
	var raceID int
	var raceName, raceDate string
	var raceRound int
	err := h.S.DB.QueryRow(`
		SELECT id, race_name, race_date, round
		FROM round_snapshots
		WHERE status = 'final'
		ORDER BY race_date DESC, round DESC
		LIMIT 1`).Scan(&raceID, &raceName, &raceDate, &raceRound)
	switch {
	case err == sql.ErrNoRows:
		// No finalized rounds: latest_race stays null.
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
		latestRace = &trmnlRace{
			Name:     raceName,
			RaceDate: raceDate,
			Round:    raceRound,
			Results:  make([]trmnlResult, 0),
		}

		// Best-effort enrichment with track metadata from the race archive.
		var country, track string
		var totalLaps int
		if err := h.S.DB.QueryRow(`
			SELECT COALESCE(country, ''), COALESCE(track, ''), COALESCE(total_laps, 0)
			FROM race_history
			WHERE name = ? AND race_date = ?
			LIMIT 1`, raceName, raceDate).Scan(&country, &track, &totalLaps); err == nil {
			latestRace.Country = country
			latestRace.Track = track
			latestRace.TotalLaps = totalLaps
		}

		rows, err := h.S.DB.Query(`
			SELECT rss.racer_name, COALESCE(t.name, ''), rss.position, rss.points,
				COALESCE(r.profile_picture, '')
			FROM round_snapshot_scores rss
			LEFT JOIN racers r ON r.id = rss.racer_id
			LEFT JOIN teams t ON t.id = r.team_id
			WHERE rss.snapshot_id = ?
			ORDER BY rss.position ASC
			LIMIT 10`, raceID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var r trmnlResult
			if err := rows.Scan(&r.RacerName, &r.Team, &r.Position, &r.Points, &r.ProfilePicture); err != nil {
				continue
			}
			r.ProfilePicture = trmnlAbsoluteURL(c, r.ProfilePicture)
			latestRace.Results = append(latestRace.Results, r)
		}
	}

	// Season: the active season, falling back to the most recently created.
	var season *trmnlSeason
	var seasonID int
	var seasonName string
	err = h.S.DB.QueryRow(`
		SELECT id, name FROM seasons
		WHERE status = 'active'
		ORDER BY id DESC
		LIMIT 1`).Scan(&seasonID, &seasonName)
	if err == sql.ErrNoRows {
		err = h.S.DB.QueryRow(`
			SELECT id, name FROM seasons
			ORDER BY id DESC
			LIMIT 1`).Scan(&seasonID, &seasonName)
	}
	switch {
	case err == sql.ErrNoRows:
		// No seasons: season stays null.
	case err != nil:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	default:
		season = &trmnlSeason{ID: seasonID, Name: seasonName}
	}

	standings := make([]models.SeasonStanding, 0)
	if season != nil {
		standings = racing.SeasonStandings(h.S.DB, seasonID, 8)
		for i := range standings {
			standings[i].ProfilePicture = trmnlAbsoluteURL(c, standings[i].ProfilePicture)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"latest_race": latestRace,
		"standings":   standings,
		"season":      season,
	})
}

// trmnlAbsoluteURL resolves a stored profile picture (usually a relative path
// like /media/.. or /static/..) into an absolute URL the TRMNL servers can
// fetch when rendering the plugin. Already-absolute URLs pass through
// unchanged.
func trmnlAbsoluteURL(c *gin.Context, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "//") {
		return path
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	}

	host := c.Request.Host
	if fwdHost := c.GetHeader("X-Forwarded-Host"); fwdHost != "" {
		host = strings.TrimSpace(strings.Split(fwdHost, ",")[0])
	}

	return scheme + "://" + host + path
}
