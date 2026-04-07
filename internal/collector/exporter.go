package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"
)

const namespace = "ch_exporter"

// Exporter implements prometheus.Collector for ClickHouse metrics.
type Exporter struct {
	cfg    *config.Config
	conn   driver.Conn
	logger *slog.Logger

	up        prometheus.Gauge
	buildInfo *prometheus.GaugeVec

	systemMetric *prometheus.GaugeVec
	systemEvent  *prometheus.GaugeVec
	asyncMetric  *prometheus.GaugeVec

	// Extended + aggressive
	replicasTotal    prometheus.Gauge
	replicasMaxDelay prometheus.Gauge
	mergesActive     prometheus.Gauge
	mutationsRunning prometheus.Gauge
	partsActive      prometheus.Gauge
	diskFreeBytes    *prometheus.GaugeVec
	diskTotalBytes   *prometheus.GaugeVec

	// Aggressive only
	partsPerTable *prometheus.GaugeVec

	scrapeErrors *prometheus.CounterVec
	scrapeDur    *prometheus.HistogramVec

	extended bool
	aggr     bool
}

// New registers metrics on reg and returns Exporter.
func New(cfg *config.Config, conn driver.Conn, logger *slog.Logger, version string, reg prometheus.Registerer) *Exporter {
	e := &Exporter{
		cfg:    cfg,
		conn:   conn,
		logger: logger,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "1 if ClickHouse responded to ping during last scrape.",
		}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "Build metadata (value is always 1).",
		}, []string{"version"}),
		systemMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "system_metric_value",
			Help:      "Value from system.metrics.",
		}, []string{"metric"}),
		systemEvent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "system_event_value",
			Help:      "Value from system.events (cumulative counter in CH).",
		}, []string{"event"}),
		asyncMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "async_metric_value",
			Help:      "Value from system.asynchronous_metrics.",
		}, []string{"metric"}),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scrape_errors_total",
			Help:      "Errors per logical step during scrape.",
		}, []string{"step"}),
		scrapeDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "scrape_step_duration_seconds",
			Help:      "Duration of scrape steps.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"step"}),
	}

	e.buildInfo.WithLabelValues(version).Set(1)

	switch cfg.Profile {
	case config.ProfileExtended, config.ProfileAggressive:
		e.extended = true
		e.replicasTotal = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "replicas_total",
			Help:      "Number of rows in system.replicas.",
		})
		e.replicasMaxDelay = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "replicas_max_absolute_delay",
			Help:      "Max absolute_delay across system.replicas.",
		})
		e.mergesActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "merges_active",
			Help:      "Active merges (rows in system.merges).",
		})
		e.mutationsRunning = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "mutations_unfinished",
			Help:      "Mutations with is_done = 0.",
		})
		e.partsActive = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "parts_active_total",
			Help:      "Active data parts (system.parts WHERE active).",
		})
		e.diskFreeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "disk_free_bytes",
			Help:      "Free space per disk from system.disks.",
		}, []string{"disk"})
		e.diskTotalBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "disk_total_bytes",
			Help:      "Total space per disk from system.disks.",
		}, []string{"disk"})
		if cfg.Profile == config.ProfileAggressive {
			e.aggr = true
			e.partsPerTable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "table_active_parts",
				Help:      "Active parts per database and table (aggressive profile, top N).",
			}, []string{"database", "table"})
		}
	}

	reg.MustRegister(e)
	return e
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.buildInfo.Describe(ch)
	e.systemMetric.Describe(ch)
	e.systemEvent.Describe(ch)
	e.asyncMetric.Describe(ch)
	if e.extended {
		e.replicasTotal.Describe(ch)
		e.replicasMaxDelay.Describe(ch)
		e.mergesActive.Describe(ch)
		e.mutationsRunning.Describe(ch)
		e.partsActive.Describe(ch)
		e.diskFreeBytes.Describe(ch)
		e.diskTotalBytes.Describe(ch)
	}
	if e.aggr {
		e.partsPerTable.Describe(ch)
	}
	e.scrapeErrors.Describe(ch)
	e.scrapeDur.Describe(ch)
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), e.cfg.CollectTimeout)
	defer cancel()

	e.systemMetric.Reset()
	e.systemEvent.Reset()
	e.asyncMetric.Reset()
	if e.extended {
		e.diskFreeBytes.Reset()
		e.diskTotalBytes.Reset()
	}
	if e.aggr {
		e.partsPerTable.Reset()
	}

	if err := e.conn.Ping(ctx); err != nil {
		e.up.Set(0)
		e.logger.Error("ping clickhouse", "err", err)
	} else {
		e.up.Set(1)
	}

	step := func(name string, fn func(context.Context) error) {
		t0 := time.Now()
		if err := fn(ctx); err != nil {
			e.scrapeErrors.WithLabelValues(name).Inc()
			e.logger.Warn("scrape step failed", "step", name, "err", err)
		}
		dt := time.Since(t0).Seconds()
		e.scrapeDur.WithLabelValues(name).Observe(dt)
	}

	step("system_metrics", e.collectSystemMetrics)
	step("system_events", e.collectSystemEvents)
	step("async_metrics", e.collectAsyncMetrics)

	if e.extended {
		step("replicas", e.collectReplicas)
		step("merges_mutations", e.collectMergesMutations)
		step("disks", e.collectDisks)
		step("parts_summary", e.collectPartsSummary)
	}
	if e.aggr {
		step("parts_top", e.collectPartsTop)
	}

	e.up.Collect(ch)
	e.buildInfo.Collect(ch)
	e.systemMetric.Collect(ch)
	e.systemEvent.Collect(ch)
	e.asyncMetric.Collect(ch)
	if e.extended {
		e.replicasTotal.Collect(ch)
		e.replicasMaxDelay.Collect(ch)
		e.mergesActive.Collect(ch)
		e.mutationsRunning.Collect(ch)
		e.partsActive.Collect(ch)
		e.diskFreeBytes.Collect(ch)
		e.diskTotalBytes.Collect(ch)
	}
	if e.aggr {
		e.partsPerTable.Collect(ch)
	}
	e.scrapeErrors.Collect(ch)
	e.scrapeDur.Collect(ch)
}
