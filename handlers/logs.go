package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// @Summary Get application logs
// @Description Get paginated application logs with optional level/module filtering
// @Tags Logs
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Entries per page (default 50)"
// @Param level query string false "Filter by level (DEBUG, INFO, WARN, ERROR)"
// @Param module query string false "Filter by module name"
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/admin/logs [get]
func (h *Handler) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	level := c.Query("level")
	module := c.Query("module")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// Build query
	where := ""
	args := make([]any, 0)
	argIdx := 0

	if level != "" {
		argIdx++
		where += " AND level = ?"
		args = append(args, level)
	}
	if module != "" {
		argIdx++
		where += " AND module = ?"
		args = append(args, module)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM app_logs WHERE 1=1" + where
	err := h.S.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		// Table may not exist yet or be empty
		total = 0
	}

	// Fetch entries
	query := "SELECT id, timestamp, level, module, message, COALESCE(data, ''), COALESCE(trace_id, '') FROM app_logs WHERE 1=1" + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, pageSize, offset)
	rows, err := h.S.DB.Query(query, queryArgs...)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"entries":  []gin.H{},
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		})
		return
	}
	defer rows.Close()

	type LogResponse struct {
		ID        int64  `json:"id"`
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Module    string `json:"module"`
		Message   string `json:"message"`
		Data      string `json:"data,omitempty"`
		TraceID   string `json:"trace_id,omitempty"`
	}

	entries := make([]LogResponse, 0)
	for rows.Next() {
		var e LogResponse
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Level, &e.Module, &e.Message, &e.Data, &e.TraceID); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"entries":  entries,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// @Summary Get log modules
// @Description Get distinct module names from the logs
// @Tags Logs
// @Produce json
// @Success 200 {array} string
// @Security cookieAuth
// @Router /api/admin/logs/modules [get]
func (h *Handler) GetLogModules(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT DISTINCT module FROM app_logs ORDER BY module")
	if err != nil {
		c.JSON(http.StatusOK, []string{})
		return
	}
	defer rows.Close()

	modules := make([]string, 0)
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			continue
		}
		modules = append(modules, m)
	}

	c.JSON(http.StatusOK, modules)
}
