package racing

import "database/sql"

// SeasonDates retrieves the start and end dates for a season.
func SeasonDates(db *sql.DB, seasonID int) (startDate, endDate string, err error) {
	err = db.QueryRow("SELECT start_date, COALESCE(end_date, '9999-12-31') FROM seasons WHERE id = ?", seasonID).
		Scan(&startDate, &endDate)
	return
}
