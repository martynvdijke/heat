package app

import "time"

// oidcPendingTTL bounds how long a completed native OIDC login stays
// claimable. The app polls every couple of seconds, so a few minutes is ample
// while keeping a leaked device code short-lived.
const oidcPendingTTL = 5 * time.Minute

// PutOIDCPendingLogin records a completed OIDC login for a device code. The
// session is handed to the app exactly once via TakeOIDCPendingLogin.
func (s *Server) PutOIDCPendingLogin(deviceCode, sessionID string) {
	s.putOIDCPendingLogin(deviceCode, OIDCPendingLogin{
		Expiry:    time.Now().Add(oidcPendingTTL).Unix(),
		SessionID: sessionID,
	})
}

// PutOIDCPendingLoginError records a failed OIDC login for a device code so the
// app can stop polling and show the reason.
func (s *Server) PutOIDCPendingLoginError(deviceCode, message string) {
	s.putOIDCPendingLogin(deviceCode, OIDCPendingLogin{
		Expiry: time.Now().Add(oidcPendingTTL).Unix(),
		Error:  message,
	})
}

func (s *Server) putOIDCPendingLogin(deviceCode string, pending OIDCPendingLogin) {
	s.OIDCPendingMu.Lock()
	defer s.OIDCPendingMu.Unlock()
	if s.OIDCPending == nil {
		s.OIDCPending = make(map[string]OIDCPendingLogin)
	}
	now := time.Now().Unix()
	for code, entry := range s.OIDCPending {
		if now > entry.Expiry {
			delete(s.OIDCPending, code)
		}
	}
	s.OIDCPending[deviceCode] = pending
}

// TakeOIDCPendingLogin returns and removes a completed login. It reports false
// while the browser flow is still pending or when the code is unknown/expired.
func (s *Server) TakeOIDCPendingLogin(deviceCode string) (OIDCPendingLogin, bool) {
	s.OIDCPendingMu.Lock()
	defer s.OIDCPendingMu.Unlock()
	pending, ok := s.OIDCPending[deviceCode]
	if !ok {
		return OIDCPendingLogin{}, false
	}
	delete(s.OIDCPending, deviceCode)
	if time.Now().Unix() > pending.Expiry {
		return OIDCPendingLogin{}, false
	}
	return pending, true
}
