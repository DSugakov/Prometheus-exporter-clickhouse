.PHONY: preflight build test lint integration-smoke metrics-snapshot baseline-compare ci docker run-safe run-extended run-aggressive
preflight:
	chmod +x ./scripts/preflight.sh
	./scripts/preflight.sh

build:
	go build -o bin/ch-exporter ./cmd/ch-exporter/

test:
	go test -race ./...

lint:
	golangci-lint run ./...

integration-smoke: preflight build
	chmod +x ./scripts/integration_smoke.sh
	./scripts/integration_smoke.sh ./bin/ch-exporter

metrics-snapshot: preflight build
	chmod +x ./scripts/collect_metrics_snapshot.sh
	./scripts/collect_metrics_snapshot.sh

baseline-compare: preflight build
	chmod +x ./scripts/baseline_compare.sh
	./scripts/baseline_compare.sh

ci: test lint integration-smoke

docker:
	docker build -t clickhouse-prometheus-exporter:local .

run-safe:
	go run ./cmd/ch-exporter -config examples/profiles/safe.yaml

run-extended:
	go run ./cmd/ch-exporter -config examples/profiles/extended.yaml

run-aggressive:
	go run ./cmd/ch-exporter -config examples/profiles/aggressive.yaml
