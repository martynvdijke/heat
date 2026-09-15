package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/app"
)

// GetRaceState returns the persisted, server-authoritative race state.
func (h *Handler) GetRaceState(c *gin.Context) {
	st, err := h.S.GetRaceState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, st)
}

// PostRaceState applies a controller lifecycle transition (start/pause/resume/
// stop), broadcasts race_state immediately, and refreshes standings.
func (h *Handler) PostRaceState(c *gin.Context) {
	var req struct {
		Action    string `json:"action"`
		TotalLaps int    `json:"total_laps"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.S.ApplyRaceAction(req.Action, req.TotalLaps)
	if err != nil {
		if errors.Is(err, app.ErrInvalidRaceState) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	app.TrySend(h.S, h.S.RaceStateBroadcast, st)
	if standings, err := h.S.ComputeStandings(0); err == nil {
		app.TrySend(h.S, h.S.StandingsBroadcast, standings)
	}
	c.JSON(http.StatusOK, st)
}
