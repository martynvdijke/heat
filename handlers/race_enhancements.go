package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// Weather

func (h *Handler) GetWeather(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	raceID, _ := strconv.Atoi(raceIDStr)

	rows, err := h.S.DB.Query("SELECT id, race_id, condition, lap_start, lap_end, grip_modifier FROM weather_conditions WHERE race_id = ? ORDER BY lap_start", raceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	weather := make([]models.WeatherCondition, 0)
	for rows.Next() {
		var w models.WeatherCondition
		if err := rows.Scan(&w.ID, &w.RaceID, &w.Condition, &w.LapStart, &w.LapEnd, &w.GripModifier); err != nil {
			continue
		}
		weather = append(weather, w)
	}
	c.JSON(http.StatusOK, weather)
}

func (h *Handler) SetWeather(c *gin.Context) {
	var w models.WeatherCondition
	if err := c.ShouldBindJSON(&w); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if w.ID == 0 {
		_, err := h.S.DB.Exec("INSERT INTO weather_conditions (race_id, condition, lap_start, lap_end, grip_modifier) VALUES (?, ?, ?, ?, ?)",
			w.RaceID, w.Condition, w.LapStart, w.LapEnd, w.GripModifier)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := h.S.DB.Exec("UPDATE weather_conditions SET condition=?, lap_start=?, lap_end=?, grip_modifier=? WHERE id=?",
			w.Condition, w.LapStart, w.LapEnd, w.GripModifier, w.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	select {
	case h.S.WeatherBroadcast <- w:
	default:
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteWeather(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid weather ID"})
		return
	}
	h.S.DB.Exec("DELETE FROM weather_conditions WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// Turbo Logs

func (h *Handler) GetTurboLogs(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	raceIDStr := c.Query("race_id")
	racerID, _ := strconv.Atoi(racerIDStr)
	raceID, _ := strconv.Atoi(raceIDStr)

	query := "SELECT id, racer_id, race_id, lap, times_used FROM turbo_logs WHERE 1=1"
	var args []interface{}
	if racerID > 0 {
		query += " AND racer_id = ?"
		args = append(args, racerID)
	}
	if raceID > 0 {
		query += " AND race_id = ?"
		args = append(args, raceID)
	}
	query += " ORDER BY racer_id, lap"

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	logs := make([]models.TurboLog, 0)
	for rows.Next() {
		var tl models.TurboLog
		if err := rows.Scan(&tl.ID, &tl.RacerID, &tl.RaceID, &tl.Lap, &tl.TimesUsed); err != nil {
			continue
		}
		logs = append(logs, tl)
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) AddTurboLog(c *gin.Context) {
	var tl models.TurboLog
	if err := c.ShouldBindJSON(&tl); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.S.DB.Exec("INSERT INTO turbo_logs (racer_id, race_id, lap, times_used) VALUES (?, ?, ?, ?)",
		tl.RacerID, tl.RaceID, tl.Lap, tl.TimesUsed)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	select {
	case h.S.GameMechanicsBroadcast <- models.GameMechanicsUpdate{
		Type: "turbo", RacerID: tl.RacerID, Action: "used",
		Data: map[string]int{"lap": tl.Lap, "times": tl.TimesUsed},
	}:
	default:
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteTurboLog(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid turbo log ID"})
		return
	}
	h.S.DB.Exec("DELETE FROM turbo_logs WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// Lap History

func (h *Handler) GetLapRecords(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	racerIDStr := c.Query("racer_id")
	raceID, _ := strconv.Atoi(raceIDStr)
	racerID, _ := strconv.Atoi(racerIDStr)

	query := "SELECT id, race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used, timestamp FROM lap_records WHERE 1=1"
	var args []interface{}
	if raceID > 0 {
		query += " AND race_id = ?"
		args = append(args, raceID)
	}
	if racerID > 0 {
		query += " AND racer_id = ?"
		args = append(args, racerID)
	}
	query += " ORDER BY race_id, lap_number, position"

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	records := make([]models.LapRecord, 0)
	for rows.Next() {
		var lr models.LapRecord
		if err := rows.Scan(&lr.ID, &lr.RaceID, &lr.RacerID, &lr.LapNumber, &lr.Position, &lr.GearUsed, &lr.HeatGenerated, &lr.TurboUsed, &lr.Timestamp); err != nil {
			continue
		}
		records = append(records, lr)
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) RecordLap(c *gin.Context) {
	var lr models.LapRecord
	if err := c.ShouldBindJSON(&lr); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.S.DB.Exec("INSERT INTO lap_records (race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used) VALUES (?, ?, ?, ?, ?, ?, ?)",
		lr.RaceID, lr.RacerID, lr.LapNumber, lr.Position, lr.GearUsed, lr.HeatGenerated, lr.TurboUsed)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Also update racer position
	h.S.DB.Exec("UPDATE racers SET position = ? WHERE id = ?", lr.Position, lr.RacerID)
	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}

func (h *Handler) RecordLapBatch(c *gin.Context) {
	var req struct {
		RaceID  int                `json:"race_id"`
		Lap     int                `json:"lap"`
		Records []models.LapRecord `json:"records"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	frame := models.LapReplayFrame{
		Type:   "lap_replay",
		RaceID: req.RaceID,
		Lap:    req.Lap,
	}

	for _, lr := range req.Records {
		lr.RaceID = req.RaceID
		lr.LapNumber = req.Lap
		_, err := h.S.DB.Exec("INSERT INTO lap_records (race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used) VALUES (?, ?, ?, ?, ?, ?, ?)",
			lr.RaceID, lr.RacerID, lr.LapNumber, lr.Position, lr.GearUsed, lr.HeatGenerated, lr.TurboUsed)
		if err != nil {
			continue
		}
		h.S.DB.Exec("UPDATE racers SET position = ? WHERE id = ?", lr.Position, lr.RacerID)

		var name, color string
		h.S.DB.QueryRow("SELECT name, car_color FROM racers WHERE id = ?", lr.RacerID).Scan(&name, &color)
		frame.Positions = append(frame.Positions, models.RacerPosition{
			RacerID: lr.RacerID, RacerName: name, Position: lr.Position,
			CarColor: color, Lap: req.Lap,
		})
	}

	select {
	case h.S.LapReplayBroadcast <- frame:
	default:
	}
	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteLapRecords(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	raceID, _ := strconv.Atoi(raceIDStr)
	if raceID > 0 {
		h.S.DB.Exec("DELETE FROM lap_records WHERE race_id = ?", raceID)
	}
	c.Status(http.StatusOK)
}

// Sectors

func (h *Handler) GetSectors(c *gin.Context) {
	trackID := c.Query("track_id")
	rows, err := h.S.DB.Query("SELECT id, name, track_id, \"order\" FROM sectors WHERE track_id = ? ORDER BY \"order\"", trackID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	sectors := make([]models.Sector, 0)
	for rows.Next() {
		var s models.Sector
		if err := rows.Scan(&s.ID, &s.Name, &s.TrackID, &s.Order); err != nil {
			continue
		}
		sectors = append(sectors, s)
	}
	c.JSON(http.StatusOK, sectors)
}

func (h *Handler) GetRacerSectors(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	racerIDStr := c.Query("racer_id")
	raceID, _ := strconv.Atoi(raceIDStr)
	racerID, _ := strconv.Atoi(racerIDStr)

	query := `SELECT rs.id, rs.race_id, rs.racer_id, rs.sector_id, rs.lap, rs.entry_time, rs.exit_time,
		s.id, s.name, s.track_id, s."order"
		FROM racer_sectors rs
		JOIN sectors s ON rs.sector_id = s.id WHERE 1=1`
	var args []interface{}
	if raceID > 0 {
		query += " AND rs.race_id = ?"
		args = append(args, raceID)
	}
	if racerID > 0 {
		query += " AND rs.racer_id = ?"
		args = append(args, racerID)
	}
	query += " ORDER BY rs.racer_id, rs.lap, s.\"order\""

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type RacerSectorView struct {
		models.RacerSector
		SectorName string `json:"sector_name"`
	}
	result := make([]RacerSectorView, 0)
	for rows.Next() {
		var rs models.RacerSector
		var s models.Sector
		if err := rows.Scan(&rs.ID, &rs.RaceID, &rs.RacerID, &rs.SectorID, &rs.Lap, &rs.EntryTime, &rs.ExitTime,
			&s.ID, &s.Name, &s.TrackID, &s.Order); err != nil {
			continue
		}
		result = append(result, RacerSectorView{RacerSector: rs, SectorName: s.Name})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) RecordRacerSector(c *gin.Context) {
	var rs models.RacerSector
	if err := c.ShouldBindJSON(&rs); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.S.DB.Exec("INSERT INTO racer_sectors (race_id, racer_id, sector_id, lap, entry_time, exit_time) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))",
		rs.RaceID, rs.RacerID, rs.SectorID, rs.Lap)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// Race Events

func (h *Handler) GetRaceEvents(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	raceID, _ := strconv.Atoi(raceIDStr)

	rows, err := h.S.DB.Query("SELECT id, race_id, lap, event_type, racer_id, racer_id2, note, timestamp FROM race_events WHERE race_id = ? ORDER BY lap, id", raceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	events := make([]models.RaceEvent, 0)
	for rows.Next() {
		var e models.RaceEvent
		if err := rows.Scan(&e.ID, &e.RaceID, &e.Lap, &e.EventType, &e.RacerID, &e.RacerID2, &e.Note, &e.Timestamp); err != nil {
			continue
		}
		events = append(events, e)
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handler) AddRaceEvent(c *gin.Context) {
	var e models.RaceEvent
	if err := c.ShouldBindJSON(&e); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.S.DB.Exec("INSERT INTO race_events (race_id, lap, event_type, racer_id, racer_id2, note) VALUES (?, ?, ?, ?, ?, ?)",
		e.RaceID, e.Lap, e.EventType, e.RacerID, e.RacerID2, e.Note)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteRaceEvent(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}
	h.S.DB.Exec("DELETE FROM race_events WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// AI Difficulty Presets (stored as settings)

func (h *Handler) GetAIDifficulty(c *gin.Context) {
	var difficulty string
	var aggression, errorRate, consistency int
	err := h.S.DB.QueryRow("SELECT COALESCE(difficulty, 'balanced'), COALESCE(aggression, 50), COALESCE(error_rate, 30), COALESCE(consistency, 50) FROM ai_settings WHERE id = 1").Scan(&difficulty, &aggression, &errorRate, &consistency)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"difficulty":  "balanced",
			"aggression":  50,
			"error_rate":  30,
			"consistency": 50,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"difficulty":  difficulty,
		"aggression":  aggression,
		"error_rate":  errorRate,
		"consistency": consistency,
	})
}

func (h *Handler) SetAIDifficulty(c *gin.Context) {
	var req struct {
		Difficulty  string `json:"difficulty"`
		Aggression  int    `json:"aggression"`
		ErrorRate   int    `json:"error_rate"`
		Consistency int    `json:"consistency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.S.DB.Exec(`INSERT INTO ai_settings (id, track_extract_url, api_key, enabled)
		VALUES (1, '', '', 0)
		ON CONFLICT(id) DO NOTHING`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = h.S.DB.Exec(`UPDATE ai_settings SET difficulty = ?, aggression = ?, error_rate = ?, consistency = ? WHERE id = 1`,
		req.Difficulty, req.Aggression, req.ErrorRate, req.Consistency)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// Sound FX
func (h *Handler) PlaySound(c *gin.Context) {
	var req struct {
		Sound string `json:"sound"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	select {
	case h.S.SoundBroadcast <- models.SoundCommand{Type: "sound", Sound: req.Sound}:
	default:
	}
	c.Status(http.StatusOK)
}
