package app

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// droppedTotal reads heat_ws_broadcast_dropped_total from the default
// registry without pulling in prometheus/testutil.
func droppedTotal(t *testing.T) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != "heat_ws_broadcast_dropped_total" {
			continue
		}
		if len(mf.GetMetric()) == 0 {
			return 0
		}
		return mf.GetMetric()[0].GetCounter().GetValue()
	}
	return 0
}

// TestTrySendCountsDrops covers the backpressure observability requirement:
// a full channel drops the message (never blocks) and increments
// heat_ws_broadcast_dropped_total; a successful send does not.
func TestTrySendCountsDrops(t *testing.T) {
	ch := make(chan int, 1)

	before := droppedTotal(t)
	TrySend[int](nil, ch, 1) // fits
	if got := droppedTotal(t); got != before {
		t.Errorf("successful send changed drop counter: got %v want %v", got, before)
	}

	TrySend[int](nil, ch, 2) // full -> dropped
	if got := droppedTotal(t); got != before+1 {
		t.Errorf("drop counter = %v, want %v", got, before+1)
	}

	if v := <-ch; v != 1 {
		t.Errorf("channel holds %d, want first (1) message only", v)
	}
}
