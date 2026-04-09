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
	Collect(ctx context.Context, conn driver.Conn, sink StepSink) error
}

type collectorStep struct {
	name      string
	min       config.Profile
	collector func(context.Context, driver.Conn, StepSink) error
}

func (s collectorStep) Name() string            { return s.name }
func (s collectorStep) MinProfile() config.Profile { return s.min }
func (s collectorStep) Collect(ctx context.Context, conn driver.Conn, sink StepSink) error {
	return s.collector(ctx, conn, sink)
}

func buildStepRegistry() []CollectorStep {
	return []CollectorStep{
		collectorStep{name: "system_metrics", min: config.ProfileSafe, collector: collectSystemMetricsStep},
		collectorStep{name: "system_events", min: config.ProfileSafe, collector: collectSystemEventsStep},
		collectorStep{name: "async_metrics", min: config.ProfileSafe, collector: collectAsyncMetricsStep},
		collectorStep{name: "replicas", min: config.ProfileExtended, collector: collectReplicasStep},
		collectorStep{name: "merges", min: config.ProfileExtended, collector: collectMergesStep},
		collectorStep{name: "mutations", min: config.ProfileExtended, collector: collectMutationsStep},
		collectorStep{name: "disks", min: config.ProfileExtended, collector: collectDisksStep},
		collectorStep{name: "parts_summary", min: config.ProfileExtended, collector: collectPartsSummaryStep},
		collectorStep{name: "demo_system_one", min: config.ProfileExtended, collector: collectDemoSystemOneStep},
		collectorStep{name: "parts_top", min: config.ProfileAggressive, collector: collectPartsTopStep},
	}
}

func selectSteps(profile config.Profile, registry []CollectorStep) []CollectorStep {
	steps := make([]CollectorStep, 0, len(registry))
	for _, step := range registry {
		if profileRank(profile) >= profileRank(step.MinProfile()) {
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
