package app

import "time"

const webSessionSchema = `CREATE TABLE IF NOT EXISTS web_sessions (
	id TEXT PRIMARY KEY,
	expiry INTEGER NOT NULL,
	ip TEXT NOT NULL
)`

// PutSession stores a web session in both the in-memory cache and SQLite so
// logins survive a restart. A nil DB (unit tests) keeps the cache-only path.
func (s *Server) PutSession(id string, info SessionInfo) {
	s.SessionStoreMu.Lock()
	s.SessionStore[id] = info
	s.SessionStoreMu.Unlock()
	if s.DB == nil {
		return
	}
	if _, err := s.DB.Exec(webSessionSchema); err != nil {
		return
	}
	_, _ = s.DB.Exec(
		`INSERT INTO web_sessions (id, expiry, ip) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET expiry=excluded.expiry, ip=excluded.ip`,
		id, info.Expiry, info.IP,
	)
}

// DeleteSession removes a web session from the cache and SQLite.
func (s *Server) DeleteSession(id string) {
	s.SessionStoreMu.Lock()
	delete(s.SessionStore, id)
	s.SessionStoreMu.Unlock()
	if s.DB == nil {
		return
	}
	_, _ = s.DB.Exec(`DELETE FROM web_sessions WHERE id = ?`, id)
}

// LookupSession resolves a session from the cache, falling back to SQLite
// after a restart. Expired rows are removed on read.
func (s *Server) LookupSession(id string) (SessionInfo, bool) {
	s.SessionStoreMu.RLock()
	info, ok := s.SessionStore[id]
	s.SessionStoreMu.RUnlock()
	if ok {
		if time.Now().Unix() > info.Expiry {
			s.DeleteSession(id)
			return SessionInfo{}, false
		}
		return info, true
	}
	if s.DB == nil {
		return SessionInfo{}, false
	}
	var stored SessionInfo
	if err := s.DB.QueryRow(`SELECT expiry, ip FROM web_sessions WHERE id = ?`, id).Scan(&stored.Expiry, &stored.IP); err != nil {
		return SessionInfo{}, false
	}
	if time.Now().Unix() > stored.Expiry {
		s.DeleteSession(id)
		return SessionInfo{}, false
	}
	s.SessionStoreMu.Lock()
	s.SessionStore[id] = stored
	s.SessionStoreMu.Unlock()
	return stored, true
}
