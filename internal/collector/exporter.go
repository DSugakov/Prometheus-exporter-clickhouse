package collector

import (
	"context"
	"log/slog"
	"strings"
	"sync"
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
	// Demonstration step metric
	demoSystemOne prometheus.Gauge

	scrapeErrors *prometheus.CounterVec
	scrapeDur    *prometheus.HistogramVec
	stepEnabled  *prometheus.GaugeVec
	stepLastOK   *prometheus.GaugeVec
	stepLastErr  *prometheus.GaugeVec

	extended bool
	aggr     bool

	stepMu        sync.RWMutex
	disabledSteps map[string]bool
	collectMu     sync.Mutex

	systemMetricFilter nameFilter
	systemEventFilter  nameFilter
	asyncMetricFilter  nameFilter
	partsDBFilter      nameFilter

	registry []CollectorStep
	steps    []CollectorStep
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
		stepEnabled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "collector_enabled",
			Help:      "1 when collector step is enabled, 0 when disabled by fail-safe.",
		}, []string{"step"}),
		stepLastOK: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "collector_last_success_unix",
			Help:      "Unix timestamp of last successful collector step run.",
		}, []string{"step"}),
		stepLastErr: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "collector_last_error_unix",
			Help:      "Unix timestamp of last failed collector step run.",
		}, []string{"step"}),
		disabledSteps: map[string]bool{},
		systemMetricFilter: newNameFilter(cfg.SystemMetricAllowlist, cfg.SystemMetricDenylist),
		systemEventFilter:  newNameFilter(cfg.SystemEventAllowlist, cfg.SystemEventDenylist),
		asyncMetricFilter:  newNameFilter(cfg.AsyncMetricAllowlist, cfg.AsyncMetricDenylist),
		partsDBFilter:      newNameFilter(cfg.PartsDatabaseAllowlist, cfg.PartsDatabaseDenylist),
		registry:           buildStepRegistry(),
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
		e.demoSystemOne = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "demo_system_one",
			Help:      "Demonstration metric from SELECT 1 on system.one.",
		})
		if cfg.Profile == config.ProfileAggressive {
			e.aggr = true
			e.partsPerTable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "table_active_parts",
				Help:      "Active parts per database and table (aggressive profile, top N).",
			}, []string{"database", "table"})
		}
	}
	e.steps = selectSteps(cfg.Profile, e.registry)

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
	if e.extended {
		e.demoSystemOne.Describe(ch)
	}
	e.scrapeErrors.Describe(ch)
	e.scrapeDur.Describe(ch)
	e.stepEnabled.Describe(ch)
	e.stepLastOK.Describe(ch)
	e.stepLastErr.Describe(ch)
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.collectMu.Lock()
	defer e.collectMu.Unlock()

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

	for _, s := range e.steps {
		local := s
		_ = e.executeStep(ctx, local.Name(), func(stepCtx context.Context) error {
			return local.Collect(stepCtx, e.conn, e)
		})
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
	if e.extended {
		e.demoSystemOne.Collect(ch)
	}
	e.scrapeErrors.Collect(ch)
	e.scrapeDur.Collect(ch)
	e.stepEnabled.Collect(ch)
	e.stepLastOK.Collect(ch)
	e.stepLastErr.Collect(ch)
}

func (e *Exporter) executeStep(ctx context.Context, name string, fn func(context.Context) error) error {
	if e.isStepDisabled(name) {
		e.stepEnabled.WithLabelValues(name).Set(0)
		return nil
	}
	e.stepEnabled.WithLabelValues(name).Set(1)

	stepCtx := ctx
	cancelStep := func() {}
	if e.cfg.QueryTimeout > 0 {
		stepCtx, cancelStep = context.WithTimeout(ctx, e.cfg.QueryTimeout)
	}
	defer cancelStep()

	t0 := time.Now()
	err := fn(stepCtx)
	if err != nil {
		e.stepLastErr.WithLabelValues(name).Set(float64(time.Now().Unix()))
		if isUnsupportedSchemaError(err) {
			e.disableStep(name, err)
			e.stepEnabled.WithLabelValues(name).Set(0)
			e.scrapeDur.WithLabelValues(name).Observe(time.Since(t0).Seconds())
			return nil
		}
		e.scrapeErrors.WithLabelValues(name).Inc()
		e.logger.Warn("scrape step failed", "step", name, "err", err)
	} else {
		e.stepLastOK.WithLabelValues(name).Set(float64(time.Now().Unix()))
	}
	e.scrapeDur.WithLabelValues(name).Observe(time.Since(t0).Seconds())
	return nil
}

func (e *Exporter) isStepDisabled(step string) bool {
	e.stepMu.RLock()
	defer e.stepMu.RUnlock()
	return e.disabledSteps[step]
}

func (e *Exporter) disableStep(step string, err error) {
	e.stepMu.Lock()
	already := e.disabledSteps[step]
	if !already {
		e.disabledSteps[step] = true
	}
	e.stepMu.Unlock()
	if already {
		return
	}
	e.logger.Warn(
		"collector step disabled due to unsupported schema",
		"step",
		step,
		"err",
		err,
	)
}

func isUnsupportedSchemaError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unknown table") ||
		strings.Contains(s, "there is no column") ||
		strings.Contains(s, "unknown identifier") ||
		strings.Contains(s, "cannot find column") ||
		strings.Contains(s, "doesn't exist")
}
