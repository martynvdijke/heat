package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// wsChannelBuffer is the buffer size for server-side WebSocket broadcast
// channels. Producers send non-blocking (see TrySend), so a saturated channel
// drops the newest message instead of stalling an HTTP handler.
const wsChannelBuffer = 64

// BroadcastDroppedTotal counts WebSocket broadcasts dropped because the
// destination buffered channel was full. Drop-newest, non-blocking by design:
// a saturated consumer must never block an HTTP handler or broadcast goroutine.
var BroadcastDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "heat_ws_broadcast_dropped_total",
	Help: "Total WebSocket broadcasts dropped because the destination channel buffer was full.",
})

// TrySend enqueues msg on ch without blocking. When the buffer is full the
// newest message is dropped, the drop counter is incremented, and a log line
// is emitted. This is the only sanctioned way for producers to feed the
// broadcast channels.
func TrySend[T any](s *Server, ch chan T, msg T) {
	select {
	case ch <- msg:
	default:
		BroadcastDroppedTotal.Inc()
		if s != nil && s.Log != nil {
			s.Log.Warnf("ws", "Broadcast dropped: channel buffer full (%T)", msg)
		}
	}
}
