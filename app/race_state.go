package app

import (
	"errors"
	"strconv"
	"time"

	"heat/models"
)

// Race lifecycle states (stored in race_state.state).
const (
	RaceStopped = "stopped"
	RaceRacing  = "racing"
	RacePaused  = "paused"
)

// ErrInvalidRaceState is returned for a transition that is not legal from the
// current state (handlers map it to 409).
var ErrInvalidRaceState = errors.New("invalid race state transition")

type raceStateRow struct {
	State         string
	StartedAt     string
	AccumulatedMs int64
	CurrentLap    int
	TotalLaps     int
}

func (s *Server) loadRaceStateRow() (raceStateRow, error) {
	var r raceStateRow
	err := s.DB.QueryRow("SELECT state, started_at, accumulated_ms, current_lap, total_laps FROM race_state WHERE id = 1").
		Scan(&r.State, &r.StartedAt, &r.AccumulatedMs, &r.CurrentLap, &r.TotalLaps)
	return r, err
}

// elapsedMs computes the authoritative elapsed time. While racing it is
// accumulated_ms plus the time since started_at; otherwise it is just
// accumulated_ms.
func (r raceStateRow) elapsedMs() int64 {
	elapsed := r.AccumulatedMs
	if r.State == RaceRacing && r.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, r.StartedAt); err == nil {
			elapsed += time.Since(t).Milliseconds()
		}
	}
	return elapsed
}

func (r raceStateRow) toModel() models.RaceState {
	return models.RaceState{
		State:      r.State,
		ElapsedMs:  r.elapsedMs(),
		CurrentLap: r.CurrentLap,
		TotalLaps:  r.TotalLaps,
	}
}

// GetRaceState returns the persisted race state with elapsed_ms computed
// server-side from monotonic time (D2).
func (s *Server) GetRaceState() (models.RaceState, error) {
	r, err := s.loadRaceStateRow()
	if err != nil {
		return models.RaceState{}, err
	}
	return r.toModel(), nil
}

// ApplyRaceAction performs a lifecycle transition, persists it, and returns the
// resulting state. Illegal transitions return ErrInvalidRaceState and leave the
// stored state untouched.
//
//	start  : stopped -> racing (resets elapsed/lap; total_laps optional)
//	pause  : racing  -> paused (folds elapsed into accumulated_ms)
//	resume : paused  -> racing
//	stop   : any     -> stopped (resets elapsed + lap)
func (s *Server) ApplyRaceAction(action string, totalLaps int) (models.RaceState, error) {
	r, err := s.loadRaceStateRow()
	if err != nil {
		return models.RaceState{}, err
	}

	now := time.Now().Format(time.RFC3339)
	switch action {
	case "start":
		if r.State != RaceStopped {
			return models.RaceState{}, ErrInvalidRaceState
		}
		r.State = RaceRacing
		r.StartedAt = now
		r.AccumulatedMs = 0
		r.CurrentLap = 1
		if totalLaps > 0 {
			r.TotalLaps = totalLaps
		}
	case "resume":
		if r.State != RacePaused {
			return models.RaceState{}, ErrInvalidRaceState
		}
		r.State = RaceRacing
		r.StartedAt = now
	case "pause":
		if r.State != RaceRacing {
			return models.RaceState{}, ErrInvalidRaceState
		}
		r.AccumulatedMs = r.elapsedMs()
		r.State = RacePaused
		r.StartedAt = ""
	case "stop":
		r.State = RaceStopped
		r.StartedAt = ""
		r.AccumulatedMs = 0
		r.CurrentLap = 0
	default:
		return models.RaceState{}, ErrInvalidRaceState
	}

	if _, err := s.DB.Exec("UPDATE race_state SET state = ?, started_at = ?, accumulated_ms = ?, current_lap = ?, total_laps = ?, updated_at = datetime('now') WHERE id = 1",
		r.State, r.StartedAt, r.AccumulatedMs, r.CurrentLap, r.TotalLaps); err != nil {
		return models.RaceState{}, err
	}
	return r.toModel(), nil
}

// ComputeStandings mirrors the controller's computeGaps semantics
// (ts/controller.ts): completed laps come from lap_records for the race; the
// leader is the racer with the highest position==1 lap (falling back to the
// most completed laps); the leader's gap is "LEAD", lapped racers show "+N",
// and same-lap racers show "". Rows are ordered by race position.
func (s *Server) ComputeStandings(raceID int) ([]models.Standing, error) {
	completed := map[int]int{}
	rows, err := s.DB.Query("SELECT racer_id, MAX(lap_number) FROM lap_records WHERE race_id = ? GROUP BY racer_id", raceID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rid, laps int
		if err := rows.Scan(&rid, &laps); err == nil {
			completed[rid] = laps
		}
	}
	rows.Close()

	leaderID, leaderLaps := -1, 0
	// Prefer the racer leading on position==1 with the most laps.
	if err := s.DB.QueryRow("SELECT racer_id, MAX(lap_number) FROM lap_records WHERE race_id = ? AND position = 1 GROUP BY racer_id ORDER BY MAX(lap_number) DESC, racer_id DESC LIMIT 1", raceID).
		Scan(&leaderID, &leaderLaps); err != nil {
		leaderID, leaderLaps = -1, 0
	}
	if leaderID == -1 {
		for rid, laps := range completed {
			if laps > leaderLaps {
				leaderLaps, leaderID = laps, rid
			}
		}
	}

	srows, err := s.DB.Query("SELECT id, name, car_color, position FROM racers ORDER BY position ASC")
	if err != nil {
		return nil, err
	}
	defer srows.Close()

	standings := []models.Standing{}
	for srows.Next() {
		var st models.Standing
		if err := srows.Scan(&st.RacerID, &st.Name, &st.CarColor, &st.Position); err != nil {
			return nil, err
		}
		st.Lap = completed[st.RacerID]
		gap := leaderLaps - st.Lap
		switch {
		case st.RacerID == leaderID:
			st.Gap = "LEAD"
		case gap >= 1:
			st.Gap = "+" + strconv.Itoa(gap)
		default:
			st.Gap = ""
		}
		standings = append(standings, st)
	}
	return standings, nil
}
