package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx := context.Background()
	conn, err := chclient.Open(ctx, cfg)
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
	go func() {
		log.Info("listening", "addr", cfg.ListenAddress, "profile", cfg.Profile)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	_ = srv.Close()
}
