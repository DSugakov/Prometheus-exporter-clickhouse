package collector

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"
)

func TestSelectStepsByProfile(t *testing.T) {
	reg := buildStepRegistry()

	safe := selectSteps(config.ProfileSafe, reg, nil, nil)
	extended := selectSteps(config.ProfileExtended, reg, nil, nil)
	aggressive := selectSteps(config.ProfileAggressive, reg, nil, nil)

	if len(safe) == 0 {
		t.Fatal("safe profile must have steps")
	}
	if len(extended) <= len(safe) {
		t.Fatalf("extended must include more steps than safe: safe=%d extended=%d", len(safe), len(extended))
	}
	if len(aggressive) <= len(extended) {
		t.Fatalf("aggressive must include more steps than extended: extended=%d aggressive=%d", len(extended), len(aggressive))
	}
}

func TestSelectStepsWithAllowDeny(t *testing.T) {
	reg := buildStepRegistry()
	got := selectSteps(config.ProfileAggressive, reg, []string{"system_metrics", "parts_top"}, []string{"parts_top"})
	if len(got) != 1 || got[0].Name() != "system_metrics" {
		t.Fatalf("unexpected filtered steps: %+v", got)
	}
}

func TestRegistryStableOrder(t *testing.T) {
	reg := buildStepRegistry()
	got := make([]string, 0, len(reg))
	for _, s := range reg {
		got = append(got, s.Name())
	}
	want := []string{
		"system_metrics",
		"system_events",
		"async_metrics",
		"replicas",
		"merges",
		"mutations",
		"disks",
		"parts_summary",
		"demo_system_one",
		"parts_top",
	}
	if len(got) != len(want) {
		t.Fatalf("registry size mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("registry order mismatch at %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func TestGracefulDisableOnUnsupportedStepError(t *testing.T) {
	e := newStepTestExporter()
	step := collectorStep{
		name: "replicas",
		min:  config.ProfileExtended,
		collector: func(context.Context, driver.Conn, StepSink) error {
			return errString("Unknown table system.replicas")
		},
	}

	// adapter since test step doesn't need conn
	err := e.executeStep(context.Background(), step.Name(), func(ctx context.Context) error {
		return step.collector(ctx, nil, e)
	})
	if err != nil {
		t.Fatalf("execute step returned error: %v", err)
	}
	if !e.isStepDisabled("replicas") {
		t.Fatal("step must be disabled after unsupported schema error")
	}
}

func TestHasRequiredSchema(t *testing.T) {
	step := collectorStep{
		name:           "parts_summary",
		min:            config.ProfileExtended,
		requiredTables: []string{"parts"},
		requiredColumns: []SchemaColumn{
			{Table: "parts", Column: "active"},
		},
	}
	ok := hasRequiredSchema(
		step,
		map[string]struct{}{"parts": {}},
		map[SchemaColumn]struct{}{{Table: "parts", Column: "active"}: {}},
	)
	if !ok {
		t.Fatal("expected schema to be available")
	}

	missingColumn := hasRequiredSchema(
		step,
		map[string]struct{}{"parts": {}},
		map[SchemaColumn]struct{}{},
	)
	if missingColumn {
		t.Fatal("expected schema check to fail when required column is missing")
	}
}

func newStepTestExporter() *Exporter {
	e := &Exporter{
		cfg: &config.Config{
			QueryTimeout: 2 * time.Second,
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "test_scrape_errors_total",
		}, []string{"step"}),
		scrapeDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "test_scrape_duration_seconds",
		}, []string{"step"}),
		stepEnabled: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "test_step_enabled",
		}, []string{"step"}),
		stepLastOK: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "test_step_last_ok",
		}, []string{"step"}),
		stepLastErr: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "test_step_last_err",
		}, []string{"step"}),
		disabledSteps: map[string]bool{},
	}
	e.timeoutPolicy = NewTimeoutPolicy(e.cfg.QueryTimeout)
	e.errorReporter = NewStepErrorReporter(e.logger, e.scrapeErrors, e.stepLastOK, e.stepLastErr)
	return e
}
