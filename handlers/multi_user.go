package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Player Session Management

func (h *Handler) PlayerLogin(c *gin.Context) {
	var req struct {
		RacerID    int    `json:"racer_id"`
		DeviceName string `json:"device_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var name string
	err := h.S.DB.QueryRow("SELECT name FROM racers WHERE id = ?", req.RacerID).Scan(&name)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Racer not found"})
		return
	}

	token := generateToken()

	// Upsert session
	_, err = h.S.DB.Exec(`INSERT INTO player_sessions (racer_id, token, device_name, last_seen, created_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(racer_id) DO UPDATE SET token = ?, device_name = ?, last_seen = datetime('now')`,
		req.RacerID, token, req.DeviceName, token, req.DeviceName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"racer_id":   req.RacerID,
		"racer_name": name,
	})
}

func (h *Handler) PlayerLogout(c *gin.Context) {
	token := c.GetHeader("X-Player-Token")
	if token == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing player token"})
		return
	}
	h.S.DB.Exec("DELETE FROM player_sessions WHERE token = ?", token)
	c.Status(http.StatusOK)
}

func (h *Handler) ValidatePlayerToken(c *gin.Context) {
	token := c.GetHeader("X-Player-Token")
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing player token"})
		return
	}

	var racerID int
	var racerName string
	var lastSeen string
	err := h.S.DB.QueryRow(`SELECT ps.racer_id, r.name, ps.last_seen
		FROM player_sessions ps
		JOIN racers r ON r.id = ps.racer_id
		WHERE ps.token = ?`, token).Scan(&racerID, &racerName, &lastSeen)

	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update last seen
	h.S.DB.Exec("UPDATE player_sessions SET last_seen = datetime('now') WHERE token = ?", token)

	c.JSON(http.StatusOK, gin.H{
		"racer_id":   racerID,
		"racer_name": racerName,
		"last_seen":  lastSeen,
	})
}

func (h *Handler) GetPlayerSessions(c *gin.Context) {
	rows, err := h.S.DB.Query(`SELECT ps.id, ps.racer_id, r.name, ps.token, ps.device_name, ps.last_seen, ps.created_at
		FROM player_sessions ps
		JOIN racers r ON r.id = ps.racer_id
		ORDER BY r.name`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type PlayerSessionView struct {
		ID         int    `json:"id"`
		RacerID    int    `json:"racer_id"`
		RacerName  string `json:"racer_name"`
		Token      string `json:"token"`
		DeviceName string `json:"device_name"`
		LastSeen   string `json:"last_seen"`
		CreatedAt  string `json:"created_at"`
	}

	sessions := make([]PlayerSessionView, 0)
	for rows.Next() {
		var s PlayerSessionView
		if err := rows.Scan(&s.ID, &s.RacerID, &s.RacerName, &s.Token, &s.DeviceName, &s.LastSeen, &s.CreatedAt); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *Handler) DeletePlayerSession(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	h.S.DB.Exec("DELETE FROM player_sessions WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// Self-Service Actions (called by players from their device)

func (h *Handler) PlayerReportGear(c *gin.Context) {
	var req struct {
		Token  string `json:"token"`
		Lap    int    `json:"lap"`
		Gear   int    `json:"gear"`
		Stress int    `json:"stress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var racerID int
	err := h.S.DB.QueryRow("SELECT racer_id FROM player_sessions WHERE token = ?", req.Token).Scan(&racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	var raceID int
	h.S.DB.QueryRow("SELECT COALESCE((SELECT id FROM race_history ORDER BY id DESC LIMIT 1), 0)").Scan(&raceID)

	_, err = h.S.DB.Exec("INSERT INTO gear_shifts (racer_id, race_id, lap, gear, stress) VALUES (?, ?, ?, ?, ?)",
		racerID, raceID, req.Lap, req.Gear, req.Stress)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	action := models.SelfServiceAction{
		Type: "gear_report", RacerID: racerID, Lap: req.Lap,
		Gear: req.Gear, Stress: req.Stress,
	}
	h.S.BroadcastSelfService(action)

	c.Status(http.StatusOK)
}

func (h *Handler) PlayerReportHeat(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		CardType string `json:"card_type"`
		Location string `json:"location"`
		Count    int    `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var racerID int
	err := h.S.DB.QueryRow("SELECT racer_id FROM player_sessions WHERE token = ?", req.Token).Scan(&racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	for i := 0; i < req.Count; i++ {
		h.S.DB.Exec("INSERT INTO heat_cards (racer_id, location, card_type, lap_added) VALUES (?, ?, ?, ?)",
			racerID, req.Location, req.CardType, 0)
	}

	select {
	case h.S.GameMechanicsBroadcast <- models.GameMechanicsUpdate{
		Type: "heat_cards", RacerID: racerID, Action: "added",
		Data: func() json.RawMessage {
			d, _ := json.Marshal(map[string]any{"count": req.Count, "location": req.Location})
			return d
		}(),
	}:
	default:
	}

	c.Status(http.StatusOK)
}

func (h *Handler) PlayerReportTurbo(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Lap   int    `json:"lap"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var racerID int
	err := h.S.DB.QueryRow("SELECT racer_id FROM player_sessions WHERE token = ?", req.Token).Scan(&racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	var raceID int
	h.S.DB.QueryRow("SELECT COALESCE((SELECT id FROM race_history ORDER BY id DESC LIMIT 1), 0)").Scan(&raceID)

	_, err = h.S.DB.Exec("INSERT INTO turbo_logs (racer_id, race_id, lap, times_used) VALUES (?, ?, ?, 1)",
		racerID, raceID, req.Lap)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	action := models.SelfServiceAction{
		Type: "turbo", RacerID: racerID, Lap: req.Lap, TurboUsed: true,
	}
	h.S.BroadcastSelfService(action)

	c.Status(http.StatusOK)
}

func (h *Handler) PlayerGetStatus(c *gin.Context) {
	token := c.GetHeader("X-Player-Token")
	if token == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing player token"})
		return
	}

	var racerID int
	err := h.S.DB.QueryRow("SELECT racer_id FROM player_sessions WHERE token = ?", token).Scan(&racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	h.S.DB.Exec("UPDATE player_sessions SET last_seen = datetime('now') WHERE token = ?", token)

	// Get racer info
	var racer models.Racer
	err = h.S.DB.QueryRow("SELECT r.id, r.name, r.profile_picture, r.car_color, r.car_name, r.points, r.rank, r.position, COALESCE(r.team_id, 0), COALESCE(t.name, '') FROM racers r LEFT JOIN teams t ON r.team_id = t.id WHERE r.id = ?",
		racerID).Scan(&racer.ID, &racer.Name, &racer.ProfilePicture, &racer.CarColor, &racer.CarName, &racer.Points, &racer.Rank, &racer.Position, &racer.TeamID, &racer.TeamName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get heat card counts
	var handCount, deckCount, discardCount, engineCount int
	h.S.DB.QueryRow("SELECT COUNT(*) FROM heat_cards WHERE racer_id = ? AND location = 'hand'", racerID).Scan(&handCount)
	h.S.DB.QueryRow("SELECT COUNT(*) FROM heat_cards WHERE racer_id = ? AND location = 'deck'", racerID).Scan(&deckCount)
	h.S.DB.QueryRow("SELECT COUNT(*) FROM heat_cards WHERE racer_id = ? AND location = 'discard'", racerID).Scan(&discardCount)
	h.S.DB.QueryRow("SELECT COUNT(*) FROM heat_cards WHERE racer_id = ? AND location = 'engine'", racerID).Scan(&engineCount)

	// Get race info
	var raceInfo models.RaceInfo
	h.S.DB.QueryRow("SELECT id, country, track, laps, track_id FROM race_info LIMIT 1").Scan(
		&raceInfo.ID, &raceInfo.Country, &raceInfo.Track, &raceInfo.Laps, &raceInfo.TrackID)

	c.JSON(http.StatusOK, gin.H{
		"racer": racer,
		"heat_cards": gin.H{
			"hand":    handCount,
			"deck":    deckCount,
			"discard": discardCount,
			"engine":  engineCount,
		},
		"race": raceInfo,
	})
}

// Spectator endpoint - public race state

func (h *Handler) GetSpectatorState(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT r.id, r.name, r.profile_picture, r.car_color, r.car_name, r.points, r.rank, r.position, COALESCE(r.team_id, 0), COALESCE(t.name, '') FROM racers r LEFT JOIN teams t ON r.team_id = t.id ORDER BY r.rank ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	racers := make([]models.Racer, 0)
	for rows.Next() {
		var r models.Racer
		rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position, &r.TeamID, &r.TeamName)
		racers = append(racers, r)
	}

	var raceInfo models.RaceInfo
	h.S.DB.QueryRow("SELECT id, country, track, laps, track_id FROM race_info LIMIT 1").Scan(
		&raceInfo.ID, &raceInfo.Country, &raceInfo.Track, &raceInfo.Laps, &raceInfo.TrackID)

	var weather models.WeatherCondition
	h.S.DB.QueryRow("SELECT id, condition, lap_start, lap_end, grip_modifier FROM weather_conditions WHERE race_id = 0 ORDER BY id DESC LIMIT 1").Scan(
		&weather.ID, &weather.Condition, &weather.LapStart, &weather.LapEnd, &weather.GripModifier)

	c.JSON(http.StatusOK, gin.H{
		"racers":  racers,
		"race":    raceInfo,
		"weather": weather,
	})
}
