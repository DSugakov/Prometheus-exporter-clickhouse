package collector

import "github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"

// StepSink is the contract for writing metrics from a CollectorStep.
type StepSink interface {
	ObserveSystemMetric(name string, value float64)
	ObserveSystemEvent(name string, value float64)
	ObserveAsyncMetric(name string, value float64)

	SetReplicas(total, maxDelay float64)
	SetMergesActive(value float64)
	SetMutationsRunning(value float64)
	SetDiskSpace(name string, free, total float64)
	SetPartsActive(value float64)
	ObserveTableActiveParts(database, table string, parts float64)
	SetDemoSystemOne(value float64)

	PartsTopN() int
}

func (e *Exporter) ObserveSystemMetric(name string, value float64) {
	if !e.systemMetricFilter.Allowed(name) {
		return
	}
	e.systemMetric.WithLabelValues(name).Set(value)
}

func (e *Exporter) ObserveSystemEvent(name string, value float64) {
	if !e.systemEventFilter.Allowed(name) {
		return
	}
	prev, ok := e.prevSystemEvents[name]
	delta := value
	if ok {
		if value >= prev {
			delta = value - prev
		} else {
			// ClickHouse server restart/reset; start a new monotonic sequence.
			delta = value
		}
	}
	e.prevSystemEvents[name] = value
	if delta > 0 {
		e.systemEvent.WithLabelValues(name).Add(delta)
	}
}

func (e *Exporter) ObserveAsyncMetric(name string, value float64) {
	if !e.asyncMetricFilter.Allowed(name) {
		return
	}
	e.asyncMetric.WithLabelValues(name).Set(value)
}

func (e *Exporter) SetReplicas(total, maxDelay float64) {
	e.replicasTotal.Set(total)
	e.replicasMaxDelay.Set(maxDelay)
}

func (e *Exporter) SetMergesActive(value float64) { e.mergesActive.Set(value) }
func (e *Exporter) SetMutationsRunning(value float64) { e.mutationsRunning.Set(value) }
func (e *Exporter) SetPartsActive(value float64) { e.partsActive.Set(value) }
func (e *Exporter) SetDemoSystemOne(value float64) { e.demoSystemOne.Set(value) }

func (e *Exporter) SetDiskSpace(name string, free, total float64) {
	e.diskFreeBytes.WithLabelValues(name).Set(free)
	e.diskTotalBytes.WithLabelValues(name).Set(total)
}

func (e *Exporter) ObserveTableActiveParts(database, table string, parts float64) {
	if !e.partsDBFilter.Allowed(database) {
		return
	}
	e.partsPerTable.WithLabelValues(database, table).Set(parts)
}

func (e *Exporter) PartsTopN() int {
	n := e.cfg.PartsTopN
	if n > config.AggressiveHardMaxPartsTopN {
		return config.AggressiveHardMaxPartsTopN
	}
	return n
}
