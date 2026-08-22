package racing

import (
	"fmt"
	"strings"
)

// SeasonFilter builds a SQL predicate constraining alias.season_id to the
// given seasons. An empty slice means "all seasons" and yields an empty
// predicate with no bind values.
func SeasonFilter(alias string, seasonIDs []int) (string, []any) {
	if len(seasonIDs) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(seasonIDs))
	args := make([]any, 0, len(seasonIDs))
	for i, id := range seasonIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	return fmt.Sprintf(" AND %s.season_id IN (%s)", alias, strings.Join(placeholders, ",")), args
}
