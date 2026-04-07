package collector

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func (e *Exporter) collectSystemMetrics(ctx context.Context) error {
	rows, err := e.conn.Query(ctx, `SELECT metric, value FROM system.metrics`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var val int64
		if err := rows.Scan(&name, &val); err != nil {
			return err
		}
		e.systemMetric.WithLabelValues(name).Set(float64(val))
	}
	return rows.Err()
}

func (e *Exporter) collectSystemEvents(ctx context.Context) error {
	rows, err := e.conn.Query(ctx, `SELECT event, value FROM system.events`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var val uint64
		if err := rows.Scan(&name, &val); err != nil {
			return err
		}
		e.systemEvent.WithLabelValues(name).Set(float64(val))
	}
	return rows.Err()
}

func (e *Exporter) collectAsyncMetrics(ctx context.Context) error {
	rows, err := e.conn.Query(ctx, `SELECT metric, value FROM system.asynchronous_metrics`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var val float64
		if err := rows.Scan(&name, &val); err != nil {
			return err
		}
		e.asyncMetric.WithLabelValues(name).Set(val)
	}
	return rows.Err()
}

func (e *Exporter) collectReplicas(ctx context.Context) error {
	var cnt, maxDelay uint64
	rows, err := e.conn.Query(ctx, `
		SELECT count(), coalesce(max(absolute_delay), 0)
		FROM system.replicas
	`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return rows.Err()
	}
	if err := rows.Scan(&cnt, &maxDelay); err != nil {
		return err
	}
	e.replicasTotal.Set(float64(cnt))
	e.replicasMaxDelay.Set(float64(maxDelay))
	return rows.Err()
}

func (e *Exporter) collectMergesMutations(ctx context.Context) error {
	var merges uint64
	if err := scanOneUint64(ctx, e.conn, `SELECT count() FROM system.merges`, &merges); err != nil {
		return err
	}
	e.mergesActive.Set(float64(merges))

	var mut uint64
	if err := scanOneUint64(ctx, e.conn, `SELECT count() FROM system.mutations WHERE is_done = 0`, &mut); err != nil {
		return err
	}
	e.mutationsRunning.Set(float64(mut))
	return nil
}

func (e *Exporter) collectDisks(ctx context.Context) error {
	rows, err := e.conn.Query(ctx, `SELECT name, free_space, total_space FROM system.disks`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var free, total uint64
		if err := rows.Scan(&name, &free, &total); err != nil {
			return err
		}
		e.diskFreeBytes.WithLabelValues(name).Set(float64(free))
		e.diskTotalBytes.WithLabelValues(name).Set(float64(total))
	}
	return rows.Err()
}

func (e *Exporter) collectPartsSummary(ctx context.Context) error {
	var n uint64
	if err := scanOneUint64(ctx, e.conn, `SELECT count() FROM system.parts WHERE active`, &n); err != nil {
		return err
	}
	e.partsActive.Set(float64(n))
	return nil
}

func (e *Exporter) collectPartsTop(ctx context.Context) error {
	q := `
		SELECT database, table, count() AS c
		FROM system.parts
		WHERE active
		GROUP BY database, table
		ORDER BY c DESC
		LIMIT ?
	`
	rows, err := e.conn.Query(ctx, q, e.cfg.PartsTopN)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var db, tbl string
		var c uint64
		if err := rows.Scan(&db, &tbl, &c); err != nil {
			return err
		}
		e.partsPerTable.WithLabelValues(db, tbl).Set(float64(c))
	}
	return rows.Err()
}

func scanOneUint64(ctx context.Context, conn driver.Conn, q string, out *uint64) error {
	rows, err := conn.Query(ctx, q)
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
