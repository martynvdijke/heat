package app

import "time"

// ValidateSession reports whether sessionID is a live, non-expired session
// valid for clientIP. It exists as a bool helper for the WebSocket handshake;
// the HTTP middleware keeps its own richer per-case error responses.
func (s *Server) ValidateSession(sessionID, clientIP string) bool {
	if sessionID == "" {
		return false
	}

	s.SessionStoreMu.RLock()
	info, ok := s.SessionStore[sessionID]
	s.SessionStoreMu.RUnlock()
	if !ok {
		return false
	}

	if time.Now().Unix() > info.Expiry {
		s.SessionStoreMu.Lock()
		delete(s.SessionStore, sessionID)
		s.SessionStoreMu.Unlock()
		return false
	}

	if info.IP != "" && clientIP != "" && info.IP != clientIP {
		return false
	}
	return true
}

// RacerIDForToken maps a player token to its bound racer id.
func (s *Server) RacerIDForToken(token string) (int, bool) {
	if token == "" {
		return 0, false
	}
	var racerID int
	if err := s.DB.QueryRow("SELECT racer_id FROM player_sessions WHERE token = ?", token).Scan(&racerID); err != nil {
		return 0, false
	}
	return racerID, true
}
