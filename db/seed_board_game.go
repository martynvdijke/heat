package db

import "log"

// SeedBoardGameTracks marks the seeded base game tracks as part of the board
// game track list. Idempotent: skips when the board_game_tracks table already
// has rows, so admins' curated list survives restarts.
//
// NOTE: the ids are collected into a slice before inserting because the test
// DB runs with a single connection (MaxOpenConns(1)); inserting while the
// SELECT rows are still open would deadlock.
func SeedBoardGameTracks() {
	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks").Scan(&count)
	if count > 0 {
		return
	}
	// Only mark tracks that exist and belong to the Base Game extension.
	rows, err := srv.DB.Query("SELECT id FROM tracks WHERE extension_id = 1 ORDER BY name")
	if err != nil {
		log.Printf("[DB] SeedBoardGameTracks: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	var inserted int
	for _, id := range ids {
		if res, err := srv.DB.Exec("INSERT OR IGNORE INTO board_game_tracks (track_id) VALUES (?)", id); err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				inserted++
			}
		}
	}
	log.Printf("[DB] Seeded %d board game tracks", inserted)
}
