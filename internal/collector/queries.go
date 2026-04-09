package collector

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func collectSystemMetricsStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	rows, err := conn.Query(ctx, `SELECT metric, value FROM system.metrics`)
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
		sink.ObserveSystemMetric(name, float64(val))
	}
	return rows.Err()
}

func collectSystemEventsStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	rows, err := conn.Query(ctx, `SELECT event, value FROM system.events`)
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
		sink.ObserveSystemEvent(name, float64(val))
	}
	return rows.Err()
}

func collectAsyncMetricsStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	rows, err := conn.Query(ctx, `SELECT metric, value FROM system.asynchronous_metrics`)
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
		sink.ObserveAsyncMetric(name, val)
	}
	return rows.Err()
}

func collectReplicasStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	var cnt, maxDelay uint64
	rows, err := conn.Query(ctx, `
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
	sink.SetReplicas(float64(cnt), float64(maxDelay))
	return rows.Err()
}

func collectMergesStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	var merges uint64
	if err := scanOneUint64(ctx, conn, `SELECT count() FROM system.merges`, &merges); err != nil {
		return err
	}
	sink.SetMergesActive(float64(merges))
	return nil
}

func collectMutationsStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	var mut uint64
	if err := scanOneUint64(ctx, conn, `SELECT count() FROM system.mutations WHERE is_done = 0`, &mut); err != nil {
		return err
	}
	sink.SetMutationsRunning(float64(mut))
	return nil
}

func collectDisksStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	rows, err := conn.Query(ctx, `SELECT name, free_space, total_space FROM system.disks`)
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
		sink.SetDiskSpace(name, float64(free), float64(total))
	}
	return rows.Err()
}

func collectPartsSummaryStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	var n uint64
	if err := scanOneUint64(ctx, conn, `SELECT count() FROM system.parts WHERE active`, &n); err != nil {
		return err
	}
	sink.SetPartsActive(float64(n))
	return nil
}

func collectPartsTopStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	q := `
		SELECT database, table, count() AS c
		FROM system.parts
		WHERE active
		GROUP BY database, table
		ORDER BY c DESC
		LIMIT ?
	`
	rows, err := conn.Query(ctx, q, sink.PartsTopN())
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
		sink.ObserveTableActiveParts(db, tbl, float64(c))
	}
	return rows.Err()
}

// collectDemoSystemOneStep is a minimal example of adding a new step via registry.
func collectDemoSystemOneStep(ctx context.Context, conn driver.Conn, sink StepSink) error {
	var one uint8
	rows, err := conn.Query(ctx, `SELECT 1 FROM system.one`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return rows.Err()
	}
	if err := rows.Scan(&one); err != nil {
		return err
	}
	sink.SetDemoSystemOne(float64(one))
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
