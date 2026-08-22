package handlers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseSeasonScope resolves the requested season scope from the query string.
//
// Priority:
//  1. season_ids=1,2,3 — comma-separated season IDs (empty segments skipped)
//  2. season_id=5      — single-season alias (backward compatible)
//  3. neither          — nil slice meaning "all seasons"
//
// Invalid or non-positive IDs return an error so callers can respond 400.
func parseSeasonScope(c *gin.Context) ([]int, error) {
	if raw := strings.TrimSpace(c.Query("season_ids")); raw != "" {
		var ids []int
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid season id %q", part)
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("no valid season ids in %q", raw)
		}
		return ids, nil
	}

	if raw := strings.TrimSpace(c.Query("season_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid season_id %q", raw)
		}
		return []int{id}, nil
	}

	return nil, nil
}

// seasonScopeCacheKey builds a cache key suffix for a parsed scope:
// ":all" when no seasons are selected, otherwise ":seasons:<sorted ids>".
func seasonScopeCacheKey(prefix string, ids []int) string {
	if len(ids) == 0 {
		return prefix + ":all"
	}
	sorted := make([]int, len(ids))
	copy(sorted, ids)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return prefix + ":seasons:" + strings.Join(parts, "-")
}
