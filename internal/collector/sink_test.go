package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestObserveSystemEventDelta(t *testing.T) {
	e := &Exporter{
		systemEvent:        prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_system_event_total"}, []string{"event"}),
		systemEventFilter:  newNameFilter(nil, nil),
		prevSystemEvents:   map[string]float64{},
	}

	e.ObserveSystemEvent("SelectQuery", 10)
	e.ObserveSystemEvent("SelectQuery", 15)

	got := testutil.ToFloat64(e.systemEvent.WithLabelValues("SelectQuery"))
	if got != 5 {
		t.Fatalf("unexpected counter value: got %v, want 5", got)
	}
}

func TestObserveSystemEventReset(t *testing.T) {
	e := &Exporter{
		systemEvent:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_system_event_total_reset"}, []string{"event"}),
		systemEventFilter: newNameFilter(nil, nil),
		prevSystemEvents:  map[string]float64{},
	}

	e.ObserveSystemEvent("InsertQuery", 100)
	e.ObserveSystemEvent("InsertQuery", 20) // reset/restart

	got := testutil.ToFloat64(e.systemEvent.WithLabelValues("InsertQuery"))
	if got != 20 {
		t.Fatalf("unexpected counter after reset: got %v, want 20", got)
	}
}

func TestObserveSystemEventFirstSampleIsBaseline(t *testing.T) {
	e := &Exporter{
		systemEvent:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_system_event_total_first"}, []string{"event"}),
		systemEventFilter: newNameFilter(nil, nil),
		prevSystemEvents:  map[string]float64{},
	}

	e.ObserveSystemEvent("Query", 42)
	got := testutil.ToFloat64(e.systemEvent.WithLabelValues("Query"))
	if got != 0 {
		t.Fatalf("first sample must be treated as baseline, got %v", got)
	}
}

func TestObserveSystemEventDenylist(t *testing.T) {
	e := &Exporter{
		systemEvent:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_system_event_total_deny"}, []string{"event"}),
		systemEventFilter: newNameFilter(nil, []string{"IgnoredEvent"}),
		prevSystemEvents:  map[string]float64{},
	}

	e.ObserveSystemEvent("IgnoredEvent", 10)
	got := testutil.ToFloat64(e.systemEvent.WithLabelValues("IgnoredEvent"))
	if got != 0 {
		t.Fatalf("denylisted event must not be exported, got %v", got)
	}
}
