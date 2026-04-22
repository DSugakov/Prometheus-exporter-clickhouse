package collector

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"
)

// CollectorStep describes an extensible scrape step.
type CollectorStep interface {
	Name() string
	MinProfile() config.Profile
	RequiredTables() []string
	RequiredColumns() []SchemaColumn
	Collect(ctx context.Context, conn driver.Conn, sink StepSink) error
}

// SchemaColumn identifies required system.<table>.<column>.
type SchemaColumn struct {
	Table  string
	Column string
}

type collectorStep struct {
	name            string
	min             config.Profile
	requiredTables  []string
	requiredColumns []SchemaColumn
	collector       func(context.Context, driver.Conn, StepSink) error
}

func (s collectorStep) Name() string {
	return s.name
}
func (s collectorStep) MinProfile() config.Profile { return s.min }
func (s collectorStep) RequiredTables() []string   { return s.requiredTables }
func (s collectorStep) RequiredColumns() []SchemaColumn {
	return s.requiredColumns
}
func (s collectorStep) Collect(ctx context.Context, conn driver.Conn, sink StepSink) error {
	return s.collector(ctx, conn, sink)
}

func buildStepRegistry() []CollectorStep {
	return []CollectorStep{
		collectorStep{name: "system_metrics", min: config.ProfileSafe, collector: collectSystemMetricsStep, requiredTables: []string{"metrics"}},
		collectorStep{
			name:            "system_events",
			min:             config.ProfileSafe,
			collector:       collectSystemEventsStep,
			requiredTables:  []string{"events"},
			requiredColumns: []SchemaColumn{{Table: "events", Column: "event"}, {Table: "events", Column: "value"}},
		},
		collectorStep{name: "async_metrics", min: config.ProfileSafe, collector: collectAsyncMetricsStep, requiredTables: []string{"asynchronous_metrics"}},
		collectorStep{
			name:            "replicas",
			min:             config.ProfileExtended,
			collector:       collectReplicasStep,
			requiredTables:  []string{"replicas"},
			requiredColumns: []SchemaColumn{{Table: "replicas", Column: "absolute_delay"}},
		},
		collectorStep{name: "merges", min: config.ProfileExtended, collector: collectMergesStep, requiredTables: []string{"merges"}},
		collectorStep{
			name:            "mutations",
			min:             config.ProfileExtended,
			collector:       collectMutationsStep,
			requiredTables:  []string{"mutations"},
			requiredColumns: []SchemaColumn{{Table: "mutations", Column: "is_done"}},
		},
		collectorStep{
			name:            "disks",
			min:             config.ProfileExtended,
			collector:       collectDisksStep,
			requiredTables:  []string{"disks"},
			requiredColumns: []SchemaColumn{{Table: "disks", Column: "free_space"}, {Table: "disks", Column: "total_space"}},
		},
		collectorStep{
			name:            "parts_summary",
			min:             config.ProfileExtended,
			collector:       collectPartsSummaryStep,
			requiredTables:  []string{"parts"},
			requiredColumns: []SchemaColumn{{Table: "parts", Column: "active"}},
		},
		collectorStep{name: "demo_system_one", min: config.ProfileExtended, collector: collectDemoSystemOneStep, requiredTables: []string{"one"}},
		collectorStep{
			name:            "parts_top",
			min:             config.ProfileAggressive,
			collector:       collectPartsTopStep,
			requiredTables:  []string{"parts"},
			requiredColumns: []SchemaColumn{{Table: "parts", Column: "database"}, {Table: "parts", Column: "table"}, {Table: "parts", Column: "active"}},
		},
	}
}

func selectSteps(profile config.Profile, registry []CollectorStep, allowlist, denylist []string) []CollectorStep {
	steps := make([]CollectorStep, 0, len(registry))
	flags := newNameFilter(allowlist, denylist)
	for _, step := range registry {
		if profileRank(profile) >= profileRank(step.MinProfile()) && flags.Allowed(step.Name()) {
			steps = append(steps, step)
		}
	}
	return steps
}

func profileRank(p config.Profile) int {
	switch p {
	case config.ProfileSafe:
		return 1
	case config.ProfileExtended:
		return 2
	case config.ProfileAggressive:
		return 3
	default:
		return 0
	}
}
