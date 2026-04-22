package main

import (
	"context"
	"flag"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/chclient"
	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/collector"
	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"
)

var version = "dev"

func main() {
	var (
		configPath = flag.String("config", "", "path to YAML config (optional)")
	)
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	cfg := config.Default()
	if *configPath != "" {
		if _, err := config.LoadFile(*configPath, cfg); err != nil {
			log.Error("load config", "err", err)
			os.Exit(1)
		}
	}
	config.ApplyEnv(cfg)
	if cfg.CollectTimeout == 0 {
		cfg.CollectTimeout = 25 * time.Second
	}
	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = 20 * time.Second
	}
	if err := cfg.Validate(); err != nil {
		log.Error("invalid config", "err", err)
		os.Exit(1)
	}
	if cfg.TLS.Insecure {
		log.Warn("TLS certificate verification is disabled (insecure_skip_verify=true)")
	}

	startupTimeout := cfg.CollectTimeout
	if cfg.QueryTimeout > startupTimeout {
		startupTimeout = cfg.QueryTimeout
	}
	if startupTimeout <= 0 {
		startupTimeout = 30 * time.Second
	}
	ctx, cancelStartup := context.WithTimeout(context.Background(), startupTimeout)
	defer cancelStartup()
	conn, err := openWithRetry(ctx, cfg, log)
	if err != nil {
		log.Error("clickhouse connect", "err", err)
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewBuildInfoCollector())
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	collector.New(cfg, conn, log, version, reg)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		rctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := chclient.Ping(rctx, conn); err != nil {
			http.Error(w, "clickhouse unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: cfg.ListenAddress, Handler: mux}
	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddress, "profile", cfg.Profile)
		serverErr <- srv.ListenAndServe()
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("http shutdown", "err", err)
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			os.Exit(1)
		}
	}
}

func openWithRetry(ctx context.Context, cfg *config.Config, log *slog.Logger) (driver.Conn, error) {
	var lastErr error
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := chclient.Open(ctx, cfg)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Warn("clickhouse not ready yet", "err", err)

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
