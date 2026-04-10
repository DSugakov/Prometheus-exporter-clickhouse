package collector

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// StepErrorReporter centralizes status/error reporting for collector steps.
type StepErrorReporter struct {
	logger      *slog.Logger
	scrapeError *prometheus.CounterVec
	stepLastOK  *prometheus.GaugeVec
	stepLastErr *prometheus.GaugeVec
}

func NewStepErrorReporter(
	logger *slog.Logger,
	scrapeError *prometheus.CounterVec,
	stepLastOK *prometheus.GaugeVec,
	stepLastErr *prometheus.GaugeVec,
) StepErrorReporter {
	return StepErrorReporter{
		logger:      logger,
		scrapeError: scrapeError,
		stepLastOK:  stepLastOK,
		stepLastErr: stepLastErr,
	}
}

func (r StepErrorReporter) OnSuccess(step string) {
	r.stepLastOK.WithLabelValues(step).Set(float64(time.Now().Unix()))
}

func (r StepErrorReporter) OnFailure(step string, err error) {
	r.stepLastErr.WithLabelValues(step).Set(float64(time.Now().Unix()))
	r.scrapeError.WithLabelValues(step).Inc()
	r.logger.Warn("scrape step failed", "step", step, "err", err)
}

func (r StepErrorReporter) OnUnsupported(step string, err error) {
	r.stepLastErr.WithLabelValues(step).Set(float64(time.Now().Unix()))
	r.logger.Warn("scrape step disabled by schema capability detection", "step", step, "err", err)
}
