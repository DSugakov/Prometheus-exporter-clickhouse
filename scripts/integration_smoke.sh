#!/usr/bin/env bash
set -euo pipefail

BIN_PATH="${1:-./bin/ch-exporter}"
EXPORTER_URL="${EXPORTER_URL:-http://127.0.0.1:9101}"
CH_ADDR="${CH_ADDR:-127.0.0.1:9000}"
PROFILE="${PROFILE:-extended}"

if [[ ! -x "$BIN_PATH" ]]; then
  echo "binary not found or not executable: $BIN_PATH"
  exit 1
fi

echo "[smoke] starting exporter against ${CH_ADDR} (${PROFILE})"
CH_EXPORTER_ADDRESS="${CH_ADDR}" \
CH_EXPORTER_USERNAME="${CH_EXPORTER_USERNAME:-default}" \
CH_EXPORTER_PASSWORD="${CH_EXPORTER_PASSWORD:-clickhouse}" \
CH_EXPORTER_PROFILE="${PROFILE}" \
CH_EXPORTER_LISTEN_ADDRESS=":9101" \
"$BIN_PATH" >/tmp/ch-exporter.log 2>&1 &
pid=$!
trap 'kill "$pid" >/dev/null 2>&1 || true' EXIT

for i in {1..30}; do
  if curl -fsS "${EXPORTER_URL}/readyz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${EXPORTER_URL}/healthz" >/dev/null
metrics="$(curl -fsS "${EXPORTER_URL}/metrics")"
grep -q '^ch_exporter_up' <<<"$metrics"
grep -q '^ch_exporter_system_metric_value' <<<"$metrics"
grep -q '^ch_exporter_scrape_step_duration_seconds' <<<"$metrics"

echo "[smoke] ok"
