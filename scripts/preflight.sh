#!/usr/bin/env bash
set -euo pipefail

echo "[preflight] checking required tools..."
command -v go >/dev/null || { echo "go not found"; exit 1; }
command -v docker >/dev/null || { echo "docker not found"; exit 1; }
command -v curl >/dev/null || { echo "curl not found"; exit 1; }

echo "[preflight] checking docker daemon..."
if ! docker info >/dev/null 2>&1; then
  echo "docker daemon is not available."
  echo "Start Docker Desktop (or daemon), then re-run:"
  echo "  docker compose up -d clickhouse"
  echo "  make integration-smoke PROFILE=extended"
  echo "  PROFILE=extended make metrics-snapshot"
  echo "  BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare"
  exit 2
fi

echo "[preflight] ok"
