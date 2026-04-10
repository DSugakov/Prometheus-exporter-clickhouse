package collector

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// QueryExecutor centralizes SQL execution primitives for collectors.
type QueryExecutor struct {
	conn driver.Conn
}

func NewQueryExecutor(conn driver.Conn) QueryExecutor {
	return QueryExecutor{conn: conn}
}

func (q QueryExecutor) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return q.conn.Query(ctx, query, args...)
}

func (q QueryExecutor) QueryOneUint64(ctx context.Context, query string, out *uint64) error {
	rows, err := q.conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return rows.Err()
	}
	if err := rows.Scan(out); err != nil {
		return err
	}
	return rows.Err()
}

// TimeoutPolicy controls per-step timeout budget.
type TimeoutPolicy struct {
	queryTimeout time.Duration
}

func NewTimeoutPolicy(queryTimeout time.Duration) TimeoutPolicy {
	return TimeoutPolicy{queryTimeout: queryTimeout}
}

func (p TimeoutPolicy) StepContext(parent context.Context) (context.Context, context.CancelFunc) {
	if p.queryTimeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, p.queryTimeout)
}
