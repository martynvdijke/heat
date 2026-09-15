package main

import (
	"testing"
	"time"

	"heat/app"
	"heat/models"
)

// TestWSRaceStateBroadcast covers the immediate transition broadcast, the 1s
// tick while racing, and event-driven standings delivery.
func TestWSRaceStateBroadcast(t *testing.T) {
	resetRaceState(t)

	url, m, srv := newAuthTestServer(t)
	go m.BroadcastRaceState()
	go m.BroadcastStandings()
	t.Cleanup(func() { srv.ApplyRaceAction("stop", 0) })

	controller := wsDial(t, url, addTestSession(t, srv))
	wsSend(t, controller, `{"type":"subscribe","topics":["race_state","standings"]}`)

	// Transition: start racing and push the new state onto the broadcast channel.
	state, err := srv.ApplyRaceAction("start", 10)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	app.TrySend(srv, srv.RaceStateBroadcast, state)

	if msg := wsWaitFor(t, controller, "race_state", 2*time.Second); wsPayload(t, msg)["state"] != "racing" {
		t.Fatalf("expected immediate racing broadcast, got %v", msg)
	}

	// While racing the manager re-broadcasts roughly every second (D5).
	if msg := wsWaitFor(t, controller, "race_state", 3*time.Second); wsPayload(t, msg)["state"] != "racing" {
		t.Fatalf("expected racing tick, got %v", msg)
	}

	// Standings are event-driven (not on the 1s tick).
	app.TrySend(srv, srv.StandingsBroadcast, []models.Standing{{RacerID: 1, Name: "A", Position: 1, Lap: 5, Gap: "LEAD"}})
	if msg := wsWaitFor(t, controller, "standings", 2*time.Second); msg["topic"] != "standings" {
		t.Fatalf("expected standings topic, got %v", msg)
	}
}

// TestWSRaceRadioDelivery covers race radio reaching a subscribed controller but
// not a public spectator.
func TestWSRaceRadioDelivery(t *testing.T) {
	url, m, srv := newAuthTestServer(t)
	go m.BroadcastRaceRadio()

	controller := wsDial(t, url, addTestSession(t, srv))
	wsSend(t, controller, `{"type":"subscribe","topics":["race_radio"]}`)

	spectator := wsDial(t, url, "")

	app.TrySend(srv, srv.RaceRadioBroadcast, models.RaceRadioMessage{ID: 1, RacerID: 7, RacerName: "Seven", Message: "Box this lap"})

	if msg := wsWaitFor(t, controller, "race_radio", 2*time.Second); wsPayload(t, msg)["message"] != "Box this lap" {
		t.Fatalf("controller did not receive race radio: %v", msg)
	}
	// wsExpectNone must be the last read on this connection.
	wsExpectNone(t, spectator, "race_radio", 300*time.Millisecond)
}
