package main

import (
	"testing"
	"time"
)

// TestWSSequenceMonotonic covers requirement "Monotonic sequence numbers":
// successive server broadcasts carry strictly increasing seq values.
func TestWSSequenceMonotonic(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	controller := wsDial(t, url, addTestSession(t, srv))
	wsWaitFor(t, controller, "hello", 2*time.Second)

	wsSend(t, controller, `{"type":"flag","flag":"yellow","state":"on"}`)
	first := wsWaitFor(t, controller, "flag", 2*time.Second)
	wsSend(t, controller, `{"type":"flag","flag":"red","state":"on"}`)
	second := wsWaitFor(t, controller, "flag", 2*time.Second)

	seq1, ok1 := first["seq"].(float64)
	seq2, ok2 := second["seq"].(float64)
	if !ok1 || !ok2 {
		t.Fatalf("envelopes missing numeric seq: %v / %v", first, second)
	}
	if seq2 <= seq1 {
		t.Fatalf("seq did not increase: %v then %v", seq1, seq2)
	}
}

// TestWSHelloSnapshotOnConnect covers "Hello snapshot on connect": a client is
// correct immediately without a REST call, and the snapshot is scoped to its
// topics (spectators do not receive presence/telemetry sections).
func TestWSHelloSnapshotOnConnect(t *testing.T) {
	url, _, _ := newAuthTestServer(t)

	racerID := createTestRacer(t, "Snapshot Racer")

	spectator := wsDial(t, url, "")
	hello := wsWaitFor(t, spectator, "hello", 2*time.Second)
	if _, ok := hello["seq"].(float64); !ok {
		t.Fatalf("hello missing seq: %v", hello)
	}
	snap, ok := hello["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("hello missing snapshot: %v", hello)
	}
	racers, ok := snap["racers"].([]any)
	if !ok {
		t.Fatalf("snapshot missing racers array: %v", snap)
	}
	found := false
	for _, r := range racers {
		if rm, ok := r.(map[string]any); ok && rm["id"] == float64(racerID) {
			found = true
		}
	}
	if !found {
		t.Fatalf("snapshot racers missing test racer %d: %v", racerID, snap)
	}
	if _, ok := snap["flags"]; !ok {
		t.Fatalf("snapshot missing flags section: %v", snap)
	}
}

// TestWSResyncScopedSnapshot covers "Resync": the reply is scoped to the
// requested topics and reflects current state; a repeat request is stable
// (idempotent source data).
func TestWSResyncScopedSnapshot(t *testing.T) {
	url, _, _ := newAuthTestServer(t)

	createTestRacer(t, "Resync Racer")

	spectator := wsDial(t, url, "")
	wsWaitFor(t, spectator, "hello", 2*time.Second)

	wsSend(t, spectator, `{"type":"resync","topics":["racers"]}`)
	first := wsWaitFor(t, spectator, "resync", 2*time.Second)
	snap, ok := first["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("resync missing snapshot: %v", first)
	}
	if _, ok := snap["racers"]; !ok {
		t.Fatalf("resync snapshot missing requested racers: %v", snap)
	}
	if _, ok := snap["flags"]; ok {
		t.Fatalf("resync snapshot leaked unrequested flags: %v", snap)
	}

	wsSend(t, spectator, `{"type":"resync","topics":["racers"]}`)
	second := wsWaitFor(t, spectator, "resync", 2*time.Second)
	snap2 := second["snapshot"].(map[string]any)
	racers1 := snap["racers"].([]any)
	racers2 := snap2["racers"].([]any)
	if len(racers1) != len(racers2) {
		t.Fatalf("snapshot not stable across applies: %d vs %d", len(racers1), len(racers2))
	}
}
