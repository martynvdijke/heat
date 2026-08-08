package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
}

// queryExtensions returns all extensions ordered for dropdowns (sort_order, name).
func (h *Handler) queryExtensions() []ExtensionSummary {
	rows, err := h.S.DB.Query("SELECT id, name, description, is_base, sort_order FROM extensions ORDER BY sort_order, name")
	if err != nil {
		return []ExtensionSummary{}
	}
	defer rows.Close()

	extensions := make([]ExtensionSummary, 0)
	for rows.Next() {
		var e ExtensionSummary
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.IsBase, &e.SortOrder); err != nil {
			continue
		}
		extensions = append(extensions, e)
	}
	return extensions
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
		(SELECT COUNT(*) FROM modules m WHERE m.extension_id = e.id) AS module_count
		FROM extensions e ORDER BY e.sort_order, e.name`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	extensions := make([]ExtensionSummary, 0)
	for rows.Next() {
		var e ExtensionSummary
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.IsBase, &e.SortOrder,
			&e.TrackCount, &e.UpgradeCount, &e.LegendCount, &e.ModuleCount); err != nil {
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

// ExtensionDetail is the full content breakdown for one extension.
type ExtensionDetail struct {
	ID          int                 `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	IsBase      int                 `json:"is_base"`
	Modules     []map[string]any    `json:"modules"`
	Tracks      []map[string]string `json:"tracks"`
	Upgrades    []map[string]string `json:"upgrades"`
	Legends     []map[string]string `json:"legends"`
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
		for rows.Next() {
			var m struct {
				ID          int
				Name        string
				Description string
				SortOrder   int
			}
			if rows.Scan(&m.ID, &m.Name, &m.Description, &m.SortOrder) == nil {
				d.Modules = append(d.Modules, map[string]any{"id": m.ID, "name": m.Name, "description": m.Description, "sort_order": m.SortOrder})
			}
		}
		rows.Close()
	}

	d.Tracks = make([]map[string]string, 0)
	rows, err = h.S.DB.Query("SELECT id, name, country FROM tracks WHERE extension_id = ? ORDER BY name", id)
	if err == nil {
		for rows.Next() {
			var t struct {
				ID, Name, Country string
			}
			if rows.Scan(&t.ID, &t.Name, &t.Country) == nil {
				d.Tracks = append(d.Tracks, map[string]string{"id": t.ID, "name": t.Name, "country": t.Country})
			}
		}
		rows.Close()
	}

	d.Upgrades = make([]map[string]string, 0)
	rows, err = h.S.DB.Query("SELECT id, name, card_type, cost FROM upgrade_cards WHERE extension_id = ? ORDER BY name", id)
	if err == nil {
		for rows.Next() {
			var u struct {
				ID       int
				Name     string
				CardType string
				Cost     int
			}
			if rows.Scan(&u.ID, &u.Name, &u.CardType, &u.Cost) == nil {
				d.Upgrades = append(d.Upgrades, map[string]string{"id": strconv.Itoa(u.ID), "name": u.Name, "card_type": u.CardType, "cost": strconv.Itoa(u.Cost)})
			}
		}
		rows.Close()
	}

	d.Legends = make([]map[string]string, 0)
	rows, err = h.S.DB.Query("SELECT id, name, ability_type, racer_name FROM legend_abilities WHERE extension_id = ? ORDER BY name", id)
	if err == nil {
		for rows.Next() {
			var l struct {
				ID          int
				Name        string
				AbilityType string
				RacerName   string
			}
			if rows.Scan(&l.ID, &l.Name, &l.AbilityType, &l.RacerName) == nil {
				d.Legends = append(d.Legends, map[string]string{"id": strconv.Itoa(l.ID), "name": l.Name, "ability_type": l.AbilityType, "racer_name": l.RacerName})
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
