package app

import (
	"net"
	"time"
)

// SameClientIP reports whether two client IPs should be treated as the same
// host for session binding. Empty values match anything. Any two loopback
// addresses match: browsers may open a WebSocket over a different loopback
// family (127.0.0.1 vs ::1) than the request that created the session, and
// both are the same machine.
func SameClientIP(a, b string) bool {
	if a == "" || b == "" {
		return true
	}
	if a == b {
		return true
	}
	ia, ib := net.ParseIP(a), net.ParseIP(b)
	if ia != nil && ib != nil {
		return ia.IsLoopback() && ib.IsLoopback()
	}
	return false
}

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

	if !SameClientIP(info.IP, clientIP) {
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
