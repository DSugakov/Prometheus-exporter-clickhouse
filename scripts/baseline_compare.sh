#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${OUT_DIR:-./artifacts/experiments}"
EXPORTER_URL="${EXPORTER_URL:-http://127.0.0.1:9101}"
BASELINE_URL="${BASELINE_URL:-http://127.0.0.1:9116/metrics}"

mkdir -p "${OUT_DIR}"
stamp="$(date +%Y%m%d_%H%M%S)"

exp_file="${OUT_DIR}/candidate_${stamp}.metrics"
base_file="${OUT_DIR}/baseline_${stamp}.metrics"
report_file="${OUT_DIR}/baseline_compare_${stamp}.md"

curl -fsS "${EXPORTER_URL}/metrics" >"${exp_file}"
curl -fsS "${BASELINE_URL}" >"${base_file}"

count_unique() {
  awk '/^[a-zA-Z_:][a-zA-Z0-9_:]*(\{.*\})?[[:space:]][^#]/{print $1}' "$1" | sed -E 's/\{.*$//' | sort -u | wc -l | tr -d ' '
}

count_series() {
  awk '/^[a-zA-Z_:][a-zA-Z0-9_:]*(\{.*\})?[[:space:]][^#]/{print}' "$1" | wc -l | tr -d ' '
}

candidate_unique="$(count_unique "${exp_file}")"
baseline_unique="$(count_unique "${base_file}")"
candidate_series="$(count_series "${exp_file}")"
baseline_series="$(count_series "${base_file}")"

{
  echo "# Baseline compare"
  echo
  echo "- candidate_url: ${EXPORTER_URL}/metrics"
  echo "- baseline_url: ${BASELINE_URL}"
  echo "- collected_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo
  echo "| Variant | Unique names | Series count |"
  echo "|---|---:|---:|"
  echo "| Candidate | ${candidate_unique} | ${candidate_series} |"
  echo "| Baseline | ${baseline_unique} | ${baseline_series} |"
} >"${report_file}"

echo "[baseline-compare] report: ${report_file}"
