package chclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_driver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/DSugakov/prometheus-exporter-clickhouse/internal/config"
)

// Open builds a clickhouse connection from config (native TCP).
func Open(_ context.Context, cfg *config.Config) (clickhouse_driver.Conn, error) {
	opts := &clickhouse.Options{
		Protocol:        clickhouse.Native,
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    cfg.MaxOpenConns,
		ConnMaxLifetime: time.Hour,
	}

	if cfg.DSN != "" {
		u, err := url.Parse(cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("parse dsn: %w", err)
		}
		host := u.Host
		if host == "" {
			return nil, fmt.Errorf("dsn: empty host")
		}
		opts.Addr = []string{host}
		opts.Auth.Database = strings.TrimPrefix(u.Path, "/")
		if opts.Auth.Database == "" {
			opts.Auth.Database = cfg.Database
		}
		if u.User != nil {
			opts.Auth.Username = u.User.Username()
			opts.Auth.Password, _ = u.User.Password()
		} else {
			opts.Auth.Username = cfg.Username
			opts.Auth.Password = cfg.Password
		}
	} else {
		if cfg.Address == "" {
			return nil, fmt.Errorf("address is required when dsn is empty")
		}
		opts.Addr = []string{cfg.Address}
		opts.Auth = clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	if cfg.TLS.Enabled || cfg.TLS.CAFile != "" || cfg.TLS.Insecure {
		tc := &tls.Config{MinVersion: tls.VersionTLS12}
		if cfg.TLS.ServerName != "" {
			tc.ServerName = cfg.TLS.ServerName
		}
		if cfg.TLS.Insecure {
			tc.InsecureSkipVerify = true
		}
		if cfg.TLS.CAFile != "" {
			pool, err := x509.SystemCertPool()
			if err != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			pem, err := os.ReadFile(cfg.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("read ca file: %w", err)
			}
			if !pool.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("append ca pem")
			}
			tc.RootCAs = pool
		}
		opts.TLS = tc
	}

	return clickhouse.Open(opts)
}

// Ping verifies connectivity.
func Ping(ctx context.Context, conn clickhouse_driver.Conn) error {
	return conn.Ping(ctx)
}
