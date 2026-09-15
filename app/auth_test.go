package app

import (
	"testing"
	"time"
)

func TestSameClientIP(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical ipv4", "127.0.0.1", "127.0.0.1", true},
		{"loopback across families", "127.0.0.1", "::1", true},
		{"ipv4-mapped loopback", "::ffff:127.0.0.1", "::1", true},
		{"empty matches anything", "", "10.0.0.5", true},
		{"distinct addresses", "10.0.0.5", "10.0.0.6", false},
		{"loopback vs remote", "127.0.0.1", "10.0.0.5", false},
	}
	for _, tc := range cases {
		if got := SameClientIP(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: SameClientIP(%q, %q) = %v, want %v", tc.name, tc.a, tc.b, got, tc.want)
		}
	}
}

// A browser may create the session over one loopback family and open the
// WebSocket over another; both are the same host and must validate.
func TestValidateSessionLoopbackFamilies(t *testing.T) {
	s := NewServer()
	s.SessionStore["sess"] = SessionInfo{Expiry: time.Now().Add(time.Hour).Unix(), IP: "127.0.0.1"}

	if !s.ValidateSession("sess", "::1") {
		t.Fatal("expected session bound to 127.0.0.1 to be valid from ::1 (same loopback host)")
	}
	if s.ValidateSession("sess", "10.1.2.3") {
		t.Fatal("expected session to be rejected from a different non-loopback IP")
	}
	if s.ValidateSession("", "127.0.0.1") {
		t.Fatal("expected empty session id to be rejected")
	}
}
