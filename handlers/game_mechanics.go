package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// Heat Cards

func GetHeatCards(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, _ := strconv.Atoi(racerIDStr)

	query := "SELECT id, racer_id, location, card_type, lap_added FROM heat_cards"
	var args []interface{}
	if racerID > 0 {
		query += " WHERE racer_id = ?"
		args = append(args, racerID)
	}
	query += " ORDER BY racer_id, location"

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	cards := make([]models.HeatCard, 0)
	for rows.Next() {
		var hc models.HeatCard
		if err := rows.Scan(&hc.ID, &hc.RacerID, &hc.Location, &hc.CardType, &hc.LapAdded); err != nil {
			continue
		}
		cards = append(cards, hc)
	}
	c.JSON(http.StatusOK, cards)
}

func AddHeatCard(c *gin.Context) {
	var hc models.HeatCard
	if err := c.ShouldBindJSON(&hc); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("INSERT INTO heat_cards (racer_id, location, card_type, lap_added) VALUES (?, ?, ?, ?)",
		hc.RacerID, hc.Location, hc.CardType, hc.LapAdded)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	select {
	case app.GameMechanicsBroadcast <- models.GameMechanicsUpdate{Type: "heat_cards", RacerID: hc.RacerID, Action: "added"}:
	default:
	}
	c.Status(http.StatusOK)
}

func MoveHeatCard(c *gin.Context) {
	var req struct {
		CardID   int    `json:"card_id"`
		Location string `json:"location"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("UPDATE heat_cards SET location = ? WHERE id = ?", req.Location, req.CardID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func DeleteHeatCard(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid card ID"})
		return
	}
	var racerID int
	app.DB.QueryRow("SELECT racer_id FROM heat_cards WHERE id = ?", id).Scan(&racerID)
	app.DB.Exec("DELETE FROM heat_cards WHERE id = ?", id)
	select {
	case app.GameMechanicsBroadcast <- (models.GameMechanicsUpdate{Type: "heat_cards", RacerID: racerID, Action: "removed"}):
	default:
	}
	c.Status(http.StatusOK)
}

func ClearHeatCards(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, _ := strconv.Atoi(racerIDStr)
	if racerID > 0 {
		app.DB.Exec("DELETE FROM heat_cards WHERE racer_id = ?", racerID)
	} else {
		app.DB.Exec("DELETE FROM heat_cards")
	}
	c.Status(http.StatusOK)
}

// Gear Shifts

func GetGearShifts(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	raceIDStr := c.Query("race_id")
	racerID, _ := strconv.Atoi(racerIDStr)
	raceID, _ := strconv.Atoi(raceIDStr)

	query := "SELECT id, racer_id, race_id, lap, gear, stress FROM gear_shifts WHERE 1=1"
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

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	shifts := make([]models.GearShift, 0)
	for rows.Next() {
		var gs models.GearShift
		if err := rows.Scan(&gs.ID, &gs.RacerID, &gs.RaceID, &gs.Lap, &gs.Gear, &gs.Stress); err != nil {
			continue
		}
		shifts = append(shifts, gs)
	}
	c.JSON(http.StatusOK, shifts)
}

func AddGearShift(c *gin.Context) {
	var gs models.GearShift
	if err := c.ShouldBindJSON(&gs); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("INSERT INTO gear_shifts (racer_id, race_id, lap, gear, stress) VALUES (?, ?, ?, ?, ?)",
		gs.RacerID, gs.RaceID, gs.Lap, gs.Gear, gs.Stress)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	select {
	case app.GameMechanicsBroadcast <- (models.GameMechanicsUpdate{
		Type: "gear_shifts", RacerID: gs.RacerID, Action: "shifted",
		Data: map[string]int{"lap": gs.Lap, "gear": gs.Gear, "stress": gs.Stress},
	}):
	default:
	}
	c.Status(http.StatusOK)
}

func DeleteGearShift(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid shift ID"})
		return
	}
	app.DB.Exec("DELETE FROM gear_shifts WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// Upgrades

func GetUpgradeCards(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, description, card_type, cost, effects FROM upgrade_cards ORDER BY card_type, cost")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	upgrades := make([]models.UpgradeCard, 0)
	for rows.Next() {
		var u models.UpgradeCard
		if err := rows.Scan(&u.ID, &u.Name, &u.Description, &u.CardType, &u.Cost, &u.Effects); err != nil {
			continue
		}
		upgrades = append(upgrades, u)
	}
	c.JSON(http.StatusOK, upgrades)
}

func SaveUpgradeCard(c *gin.Context) {
	var u models.UpgradeCard
	if err := c.ShouldBindJSON(&u); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if u.ID == 0 {
		_, err := app.DB.Exec("INSERT INTO upgrade_cards (name, description, card_type, cost, effects) VALUES (?, ?, ?, ?, ?)",
			u.Name, u.Description, u.CardType, u.Cost, u.Effects)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE upgrade_cards SET name=?, description=?, card_type=?, cost=?, effects=? WHERE id=?",
			u.Name, u.Description, u.CardType, u.Cost, u.Effects, u.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)
}

func DeleteUpgradeCard(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid upgrade ID"})
		return
	}
	app.DB.Exec("DELETE FROM upgrade_cards WHERE id = ?", id)
	app.DB.Exec("DELETE FROM player_upgrades WHERE upgrade_id = ?", id)
	c.Status(http.StatusOK)
}

// Player Upgrades

func GetPlayerUpgrades(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	seasonIDStr := c.Query("season_id")
	racerID, _ := strconv.Atoi(racerIDStr)
	seasonID, _ := strconv.Atoi(seasonIDStr)

	query := `SELECT pu.id, pu.racer_id, pu.upgrade_id, pu.season_id, pu.equipped, pu.round_bought,
		uc.id, uc.name, uc.description, uc.card_type, uc.cost, uc.effects
		FROM player_upgrades pu
		JOIN upgrade_cards uc ON pu.upgrade_id = uc.id WHERE 1=1`
	var args []interface{}
	if racerID > 0 {
		query += " AND pu.racer_id = ?"
		args = append(args, racerID)
	}
	if seasonID > 0 {
		query += " AND pu.season_id = ?"
		args = append(args, seasonID)
	}
	query += " ORDER BY pu.racer_id, pu.round_bought"

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	playerUpgrades := make([]models.PlayerUpgrade, 0)
	for rows.Next() {
		var pu models.PlayerUpgrade
		var uc models.UpgradeCard
		if err := rows.Scan(&pu.ID, &pu.RacerID, &pu.UpgradeID, &pu.SeasonID, &pu.Equipped, &pu.RoundBought,
			&uc.ID, &uc.Name, &uc.Description, &uc.CardType, &uc.Cost, &uc.Effects); err != nil {
			continue
		}
		pu.Upgrade = &uc
		playerUpgrades = append(playerUpgrades, pu)
	}
	c.JSON(http.StatusOK, playerUpgrades)
}

func BuyUpgrade(c *gin.Context) {
	var req struct {
		RacerID   int `json:"racer_id"`
		UpgradeID int `json:"upgrade_id"`
		SeasonID  int `json:"season_id"`
		Round     int `json:"round"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("INSERT INTO player_upgrades (racer_id, upgrade_id, season_id, equipped, round_bought) VALUES (?, ?, ?, 1, ?)",
		req.RacerID, req.UpgradeID, req.SeasonID, req.Round)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	select {
	case app.GameMechanicsBroadcast <- (models.GameMechanicsUpdate{
		Type: "upgrades", RacerID: req.RacerID, Action: "bought",
		Data: map[string]int{"upgrade_id": req.UpgradeID, "round": req.Round},
	}):
	default:
	}
	c.Status(http.StatusOK)
}

func ToggleUpgrade(c *gin.Context) {
	var req struct {
		ID       int  `json:"id"`
		Equipped bool `json:"equipped"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	equipped := 0
	if req.Equipped {
		equipped = 1
	}
	_, err := app.DB.Exec("UPDATE player_upgrades SET equipped = ? WHERE id = ?", equipped, req.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func DeletePlayerUpgrade(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid upgrade ID"})
		return
	}
	app.DB.Exec("DELETE FROM player_upgrades WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// Legend Abilities

func GetLegendAbilities(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, description, ability_type, racer_name FROM legend_abilities")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	abilities := make([]models.LegendAbility, 0)
	for rows.Next() {
		var la models.LegendAbility
		if err := rows.Scan(&la.ID, &la.Name, &la.Description, &la.AbilityType, &la.RacerName); err != nil {
			continue
		}
		abilities = append(abilities, la)
	}
	c.JSON(http.StatusOK, abilities)
}

func GetRacerLegendAbilities(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, _ := strconv.Atoi(racerIDStr)

	query := `SELECT rla.id, rla.racer_id, rla.ability_id, rla.active,
		la.id, la.name, la.description, la.ability_type, la.racer_name
		FROM racer_legend_abilities rla
		JOIN legend_abilities la ON rla.ability_id = la.id`
	var args []interface{}
	if racerID > 0 {
		query += " WHERE rla.racer_id = ?"
		args = append(args, racerID)
	}

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := make([]models.RacerLegendAbility, 0)
	for rows.Next() {
		var rla models.RacerLegendAbility
		var la models.LegendAbility
		if err := rows.Scan(&rla.ID, &rla.RacerID, &rla.AbilityID, &rla.Active,
			&la.ID, &la.Name, &la.Description, &la.AbilityType, &la.RacerName); err != nil {
			continue
		}
		rla.Ability = &la
		result = append(result, rla)
	}
	c.JSON(http.StatusOK, result)
}

func AssignLegendAbility(c *gin.Context) {
	var req struct {
		RacerID   int `json:"racer_id"`
		AbilityID int `json:"ability_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("INSERT OR IGNORE INTO racer_legend_abilities (racer_id, ability_id, active) VALUES (?, ?, 1)",
		req.RacerID, req.AbilityID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func ToggleLegendAbility(c *gin.Context) {
	var req struct {
		ID     int  `json:"id"`
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	active := 0
	if req.Active {
		active = 1
	}
	_, err := app.DB.Exec("UPDATE racer_legend_abilities SET active = ? WHERE id = ?", active, req.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// Deck Builder

func GetAvailableUpgradesForRacer(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, _ := strconv.Atoi(racerIDStr)

	rows, err := app.DB.Query(`SELECT uc.id, uc.name, uc.description, uc.card_type, uc.cost, uc.effects
		FROM upgrade_cards uc WHERE uc.card_type = 'upgrade'
		AND uc.id NOT IN (
			SELECT upgrade_id FROM player_upgrades WHERE racer_id = ?
		) ORDER BY uc.cost`, racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	upgrades := make([]models.UpgradeCard, 0)
	for rows.Next() {
		var u models.UpgradeCard
		if err := rows.Scan(&u.ID, &u.Name, &u.Description, &u.CardType, &u.Cost, &u.Effects); err != nil {
			continue
		}
		upgrades = append(upgrades, u)
	}
	c.JSON(http.StatusOK, upgrades)
}

// Bulk heat card operations for race start
func InitializeHeatDecks(c *gin.Context) {
	var req struct {
		RaceID   int   `json:"race_id"`
		RacerIDs []int `json:"racer_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, racerID := range req.RacerIDs {
		// Each racer starts with 7 heat cards in their deck
		for i := 0; i < 7; i++ {
			app.DB.Exec("INSERT INTO heat_cards (racer_id, location, card_type, lap_added) VALUES (?, 'deck', 'heat', 0)",
				racerID)
		}
		// Deal 3 to hand
		app.DB.Exec(`UPDATE heat_cards SET location = 'hand' WHERE racer_id = ? AND location = 'deck' LIMIT 3`, racerID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Decks initialized"})
}
