#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${OUT_DIR:-./artifacts/experiments}"
EXPORTER_URL="${EXPORTER_URL:-http://127.0.0.1:9101}"
PROFILE="${PROFILE:-extended}"

mkdir -p "${OUT_DIR}"
stamp="$(date +%Y%m%d_%H%M%S)"
metrics_file="${OUT_DIR}/exporter_${PROFILE}_${stamp}.metrics"
summary_file="${OUT_DIR}/exporter_${PROFILE}_${stamp}.summary.txt"

curl -fsS "${EXPORTER_URL}/metrics" >"${metrics_file}"

series_count="$(awk '/^[a-zA-Z_:][a-zA-Z0-9_:]*(\{.*\})?[[:space:]][^#]/{print}' "${metrics_file}" | wc -l | tr -d ' ')"
unique_names="$(awk '/^[a-zA-Z_:][a-zA-Z0-9_:]*(\{.*\})?[[:space:]][^#]/{print $1}' "${metrics_file}" | sed -E 's/\{.*$//' | sort -u | wc -l | tr -d ' ')"

{
  echo "profile=${PROFILE}"
  echo "metrics_file=${metrics_file}"
  echo "unique_metric_names=${unique_names}"
  echo "series_count=${series_count}"
  echo "collected_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} >"${summary_file}"

echo "[snapshot] metrics: ${metrics_file}"
echo "[snapshot] summary: ${summary_file}"
