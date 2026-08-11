package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/ent/track"
	"heat/models"
)

// ExtensionSummary is a compact extension row with content counts for the tracker UI.
type ExtensionSummary struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsBase       int    `json:"is_base"`
	SortOrder    int    `json:"sort_order"`
	TrackCount   int    `json:"track_count"`
	UpgradeCount int    `json:"upgrade_count"`
	LegendCount  int    `json:"legend_count"`
	ModuleCount  int    `json:"module_count"`
	Owned        bool   `json:"owned"`
}

// queryExtensions returns all extensions ordered for dropdowns (sort_order, name).
func (h *Handler) queryExtensions() []ExtensionSummary {
	rows, err := h.S.DB.Query(`SELECT e.id, e.name, e.description, e.is_base, e.sort_order,
		(oe.extension_id IS NOT NULL) AS owned
		FROM extensions e LEFT JOIN owned_extensions oe ON oe.extension_id = e.id
		ORDER BY e.sort_order, e.name`)
	if err != nil {
		return []ExtensionSummary{}
	}
	defer rows.Close()

	extensions := make([]ExtensionSummary, 0)
	for rows.Next() {
		var e ExtensionSummary
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.IsBase, &e.SortOrder, &e.Owned); err != nil {
			continue
		}
		extensions = append(extensions, e)
	}
	return extensions
}

// queryModuleTracks returns the full track shape for the tracks attributed to a module.
func (h *Handler) queryModuleTracks(moduleID int) []models.Track {
	tracks := make([]models.Track, 0)
	entTracks, err := h.S.Ent.Track.Query().Where(track.ModuleID(moduleID)).Order(track.ByName()).All(context.Background())
	if err != nil {
		return tracks
	}
	boardGame := h.boardGameTrackSet()
	for _, t := range entTracks {
		tracks = append(tracks, trackToModel(t, boardGame))
	}
	return tracks
}

// @Summary List extensions
// @Description List all extensions with content counts (tracks, upgrades, legends, modules)
// @Tags Extensions
// @Produce json
// @Success 200 {array} ExtensionSummary
// @Security cookieAuth
// @Router /api/extensions [get]
func (h *Handler) GetExtensions(c *gin.Context) {
	rows, err := h.S.DB.Query(`SELECT e.id, e.name, e.description, e.is_base, e.sort_order,
		(SELECT COUNT(*) FROM tracks t WHERE t.extension_id = e.id) AS track_count,
		(SELECT COUNT(*) FROM upgrade_cards u WHERE u.extension_id = e.id) AS upgrade_count,
		(SELECT COUNT(*) FROM legend_abilities l WHERE l.extension_id = e.id) AS legend_count,
		(SELECT COUNT(*) FROM modules m WHERE m.extension_id = e.id) AS module_count,
		(oe.extension_id IS NOT NULL) AS owned
		FROM extensions e LEFT JOIN owned_extensions oe ON oe.extension_id = e.id
		ORDER BY e.sort_order, e.name`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	extensions := make([]ExtensionSummary, 0)
	for rows.Next() {
		var e ExtensionSummary
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.IsBase, &e.SortOrder,
			&e.TrackCount, &e.UpgradeCount, &e.LegendCount, &e.ModuleCount, &e.Owned); err != nil {
			continue
		}
		extensions = append(extensions, e)
	}
	c.JSON(http.StatusOK, extensions)
}

// @Summary Create extension
// @Description Create a new extension pack
// @Tags Extensions
// @Accept json
// @Produce json
// @Param body body object true "Extension" SchemaExample({"name":"My Pack","description":"...","is_base":0,"sort_order":3})
// @Success 200 {object} ExtensionSummary
// @Security cookieAuth
// @Router /api/extensions [post]
func (h *Handler) CreateExtension(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsBase      int    `json:"is_base"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	res, err := h.S.DB.Exec("INSERT INTO extensions (name, description, is_base, sort_order) VALUES (?, ?, ?, ?)",
		input.Name, input.Description, input.IsBase, input.SortOrder)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// @Summary Update extension
// @Description Update an extension's fields
// @Tags Extensions
// @Accept json
// @Security cookieAuth
// @Router /api/extensions [put]
func (h *Handler) UpdateExtension(c *gin.Context) {
	var input struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IsBase      int    `json:"is_base"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	if input.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	_, err := h.S.DB.Exec("UPDATE extensions SET name=?, description=?, is_base=?, sort_order=? WHERE id=?",
		input.Name, input.Description, input.IsBase, input.SortOrder, input.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// @Summary Delete extension
// @Description Delete an extension; its content is reset to the Base Game extension and its modules (and season module links) are removed
// @Tags Extensions
// @Param id query string true "Extension ID"
// @Success 200
// @Security cookieAuth
// @Router /api/extensions [delete]
func (h *Handler) DeleteExtension(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	// Reset content to base game (id 1)
	h.S.DB.Exec("UPDATE tracks SET extension_id = 1 WHERE extension_id = ?", id)
	h.S.DB.Exec("UPDATE upgrade_cards SET extension_id = 1 WHERE extension_id = ?", id)
	h.S.DB.Exec("UPDATE legend_abilities SET extension_id = 1 WHERE extension_id = ?", id)
	// Reset tracks owned by this extension's modules
	h.S.DB.Exec("UPDATE tracks SET module_id = 0, extension_id = 1 WHERE module_id IN (SELECT id FROM modules WHERE extension_id = ?)", id)
	// Remove season module links for the extension's modules, then the modules
	h.S.DB.Exec("DELETE FROM season_modules WHERE module_id IN (SELECT id FROM modules WHERE extension_id = ?)", id)
	h.S.DB.Exec("DELETE FROM modules WHERE extension_id = ?", id)
	// Finally remove the extension itself
	if _, err := h.S.DB.Exec("DELETE FROM extensions WHERE id = ?", id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// ExtensionDetail is the full content breakdown for one extension. Content uses
// the same shapes as the rest of the site (models.Track, models.UpgradeCard,
// models.LegendAbility) so the catalog can drive selection lists directly.
type ExtensionDetail struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	IsBase      int                    `json:"is_base"`
	Modules     []map[string]any       `json:"modules"`
	Tracks      []models.Track         `json:"tracks"`
	Upgrades    []models.UpgradeCard   `json:"upgrades"`
	Legends     []models.LegendAbility `json:"legends"`
}

// @Summary Extension detail
// @Description Get one extension with its modules, tracks, upgrade cards, and legend abilities
// @Tags Extensions
// @Produce json
// @Param id query string true "Extension ID"
// @Success 200 {object} ExtensionDetail
// @Security cookieAuth
// @Router /api/extensions/detail [get]
func (h *Handler) GetExtensionDetail(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	var d ExtensionDetail
	if err := h.S.DB.QueryRow("SELECT id, name, description, is_base FROM extensions WHERE id = ?", id).
		Scan(&d.ID, &d.Name, &d.Description, &d.IsBase); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "extension not found"})
		return
	}

	d.Modules = make([]map[string]any, 0)
	rows, err := h.S.DB.Query("SELECT id, name, description, sort_order FROM modules WHERE extension_id = ? ORDER BY sort_order", id)
	if err == nil {
		type moduleRow struct {
			ID          int
			Name        string
			Description string
			SortOrder   int
		}
		var mods []moduleRow
		for rows.Next() {
			var m moduleRow
			if rows.Scan(&m.ID, &m.Name, &m.Description, &m.SortOrder) == nil {
				mods = append(mods, m)
			}
		}
		rows.Close()
		// Collect rows before querying per-module tracks: the test DB runs with
		// a single connection, so nested queries would deadlock on open rows.
		for _, m := range mods {
			mod := map[string]any{"id": m.ID, "name": m.Name, "description": m.Description, "sort_order": m.SortOrder}
			mod["tracks"] = h.queryModuleTracks(m.ID)
			d.Modules = append(d.Modules, mod)
		}
	}

	// Full content shapes (models.Track / UpgradeCard / LegendAbility) so the
	// catalog can drive selection lists with the same data the rest of the site uses.
	boardGame := h.boardGameTrackSet()
	entTracks, err := h.S.Ent.Track.Query().Where(track.ExtensionID(id)).Order(track.ByName()).All(c.Request.Context())
	if err == nil {
		d.Tracks = make([]models.Track, 0, len(entTracks))
		for _, t := range entTracks {
			d.Tracks = append(d.Tracks, trackToModel(t, boardGame))
		}
	}

	d.Upgrades = make([]models.UpgradeCard, 0)
	rows, err = h.S.DB.Query("SELECT id, name, description, card_type, cost, effects, extension_id FROM upgrade_cards WHERE extension_id = ? ORDER BY name", id)
	if err == nil {
		for rows.Next() {
			var u models.UpgradeCard
			if rows.Scan(&u.ID, &u.Name, &u.Description, &u.CardType, &u.Cost, &u.Effects, &u.ExtensionID) == nil {
				d.Upgrades = append(d.Upgrades, u)
			}
		}
		rows.Close()
	}

	d.Legends = make([]models.LegendAbility, 0)
	rows, err = h.S.DB.Query("SELECT id, name, description, ability_type, racer_name, extension_id FROM legend_abilities WHERE extension_id = ? ORDER BY name", id)
	if err == nil {
		for rows.Next() {
			var l models.LegendAbility
			if rows.Scan(&l.ID, &l.Name, &l.Description, &l.AbilityType, &l.RacerName, &l.ExtensionID) == nil {
				d.Legends = append(d.Legends, l)
			}
		}
		rows.Close()
	}

	c.JSON(http.StatusOK, d)
}

// ModuleSummary is a module row with its owning extension's name.
type ModuleSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ExtensionID int    `json:"extension_id"`
	Extension   string `json:"extension"`
	SortOrder   int    `json:"sort_order"`
}

// queryModules returns all modules with their owning extension's name.
func (h *Handler) queryModules() ([]ModuleSummary, error) {
	rows, err := h.S.DB.Query(`SELECT m.id, m.name, m.description, m.extension_id, m.sort_order, COALESCE(e.name, '')
		FROM modules m LEFT JOIN extensions e ON m.extension_id = e.id
		ORDER BY m.sort_order, m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	modules := make([]ModuleSummary, 0)
	for rows.Next() {
		var m ModuleSummary
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.ExtensionID, &m.SortOrder, &m.Extension); err != nil {
			continue
		}
		modules = append(modules, m)
	}
	return modules, nil
}

// @Summary List modules
// @Description List all gameplay modules with their owning extension
// @Tags Extensions
// @Produce json
// @Success 200 {array} ModuleSummary
// @Security cookieAuth
// @Router /api/modules [get]
func (h *Handler) GetModules(c *gin.Context) {
	modules, err := h.queryModules()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, modules)
}

// @Summary Create module
// @Description Create a gameplay module, optionally owned by an extension
// @Tags Extensions
// @Accept json
// @Security cookieAuth
// @Router /api/modules [post]
func (h *Handler) CreateModule(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ExtensionID int    `json:"extension_id"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	res, err := h.S.DB.Exec("INSERT INTO modules (name, description, extension_id, sort_order) VALUES (?, ?, ?, ?)",
		input.Name, input.Description, input.ExtensionID, input.SortOrder)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// @Summary Update module
// @Description Update a gameplay module
// @Tags Extensions
// @Accept json
// @Security cookieAuth
// @Router /api/modules [put]
func (h *Handler) UpdateModule(c *gin.Context) {
	var input struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		ExtensionID int    `json:"extension_id"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	if input.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	_, err := h.S.DB.Exec("UPDATE modules SET name=?, description=?, extension_id=?, sort_order=? WHERE id=?",
		input.Name, input.Description, input.ExtensionID, input.SortOrder, input.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// @Summary Delete module
// @Description Delete a gameplay module and its season links
// @Tags Extensions
// @Param id query string true "Module ID"
// @Success 200
// @Security cookieAuth
// @Router /api/modules [delete]
func (h *Handler) DeleteModule(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	h.S.DB.Exec("DELETE FROM season_modules WHERE module_id = ?", id)
	// Reset tracks owned by this module to the Base Game extension
	h.S.DB.Exec("UPDATE tracks SET module_id = 0, extension_id = 1 WHERE module_id = ?", id)
	if _, err := h.S.DB.Exec("DELETE FROM modules WHERE id = ?", id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// @Summary Assign content to extension
// @Description Assign a track, upgrade card, or legend ability to an extension
// @Tags Extensions
// @Accept json
// @Param body body object true "Assignment" SchemaExample({"content_type":"track","content_id":"monza","extension_id":2})
// @Success 200
// @Security cookieAuth
// @Router /api/content/extension [put]
func (h *Handler) AssignContentExtension(c *gin.Context) {
	var input struct {
		ContentType string `json:"content_type"` // track | upgrade | legend
		ContentID   string `json:"content_id"`
		ExtensionID int    `json:"extension_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.ContentType == "" || input.ContentID == "" || input.ExtensionID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "content_type, content_id, extension_id required"})
		return
	}
	// Validate extension exists
	var count int
	if err := h.S.DB.QueryRow("SELECT COUNT(*) FROM extensions WHERE id = ?", input.ExtensionID).Scan(&count); err != nil || count == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "extension not found"})
		return
	}

	var err error
	switch input.ContentType {
	case "track":
		_, err = h.S.DB.Exec("UPDATE tracks SET extension_id = ? WHERE id = ?", input.ExtensionID, input.ContentID)
	case "upgrade":
		_, err = h.S.DB.Exec("UPDATE upgrade_cards SET extension_id = ? WHERE id = ?", input.ExtensionID, input.ContentID)
	case "legend":
		_, err = h.S.DB.Exec("UPDATE legend_abilities SET extension_id = ? WHERE id = ?", input.ExtensionID, input.ContentID)
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "content_type must be track, upgrade, or legend"})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// baseExtensionID returns the id of the Base Game extension (is_base = 1),
// defaulting to 1 when no extension is flagged as base.
func (h *Handler) baseExtensionID() int {
	var id int
	if err := h.S.DB.QueryRow("SELECT id FROM extensions WHERE is_base = 1 ORDER BY id LIMIT 1").Scan(&id); err != nil || id <= 0 {
		return 1
	}
	return id
}

// ownedExtensionIDs returns the owned extension ids plus 0. Content with
// extension_id = 0 is normalized to the Base Game and always selectable.
func (h *Handler) ownedExtensionIDs() []int {
	ids := []int{0}
	seen := map[int]bool{0: true}
	rows, err := h.S.DB.Query("SELECT extension_id FROM owned_extensions")
	if err != nil {
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if rows.Scan(&id) == nil && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// @Summary List owned extensions
// @Description Return the ids of the extensions the group owns; the Base Game is always included
// @Tags Extensions
// @Produce json
// @Success 200 {object} map[string][]int
// @Security cookieAuth
// @Router /api/extensions/owned [get]
func (h *Handler) GetOwnedExtensions(c *gin.Context) {
	ids := make([]int, 0)
	seen := map[int]bool{}
	rows, err := h.S.DB.Query("SELECT extension_id FROM owned_extensions")
	if err == nil {
		for rows.Next() {
			var id int
			if rows.Scan(&id) == nil && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		rows.Close()
	}
	base := h.baseExtensionID()
	if !seen[base] {
		ids = append(ids, base)
	}
	c.JSON(http.StatusOK, gin.H{"owned_ids": ids})
}

// @Summary Set owned extensions
// @Description Full-replace the owned extension set; the Base Game is always re-added and unknown ids are ignored
// @Tags Extensions
// @Accept json
// @Security cookieAuth
// @Router /api/extensions/owned [put]
func (h *Handler) SetOwnedExtensions(c *gin.Context) {
	var input struct {
		OwnedIDs []int `json:"owned_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	tx, err := h.S.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM owned_extensions"); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	seen := map[int]bool{}
	for _, id := range input.OwnedIDs {
		if id <= 0 || seen[id] {
			continue
		}
		// Ignore unknown ids so a stale editor can't reference deleted extensions.
		var exists int
		if err := tx.QueryRow("SELECT COUNT(*) FROM extensions WHERE id = ?", id).Scan(&exists); err != nil || exists == 0 {
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO owned_extensions (extension_id) VALUES (?)", id); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		seen[id] = true
	}
	// The Base Game is always owned, regardless of what was submitted.
	// Resolve it on the tx so we don't grab a second connection (the test DB
	// runs with a single connection and would deadlock on the tx below).
	var base int
	if err := tx.QueryRow("SELECT id FROM extensions WHERE is_base = 1 ORDER BY id LIMIT 1").Scan(&base); err != nil || base <= 0 {
		base = 1
	}
	if !seen[base] {
		if _, err := tx.Exec("INSERT OR IGNORE INTO owned_extensions (extension_id) VALUES (?)", base); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
