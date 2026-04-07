# Prometheus-exporter-clickhouse

Репозиторий: `git@github.com:DSugakov/Prometheus-exporter-clickhouse.git`

Внешний Prometheus-exporter для ClickHouse на Go: опрос `system.*` по SQL **без** изменения server-side конфигурации ClickHouse.

## Быстрый старт

```bash
export CH_EXPORTER_ADDRESS=localhost:9000
export CH_EXPORTER_USERNAME=default
export CH_EXPORTER_PASSWORD=clickhouse
export CH_EXPORTER_PROFILE=safe
go run ./cmd/ch-exporter/
```

Эндпоинты:

- `GET /metrics` — Prometheus
- `GET /healthz` — liveness
- `GET /readyz` — readiness (ping ClickHouse)

## Профили

| Значение `CH_EXPORTER_PROFILE` | Коллекторы |
|-------------------------------|------------|
| `safe` | `system.metrics`, `system.events`, `system.asynchronous_metrics` |
| `extended` | + реплики, merges/mutations, диски, сводка parts |
| `aggressive` | + top-N таблиц по числу активных кусков (`parts_top_n`) |

## Конфигурация

- Файл: `ch-exporter -config examples/config.yaml`
- Переменные окружения с префиксом `CH_EXPORTER_`: см. [internal/config/config.go](internal/config/config.go) (`LISTEN_ADDRESS`, `DSN`, `ADDRESS`, `PROFILE`, `PARTS_TOP_N`, TLS).

Пример DSN: `clickhouse://default@localhost:9000/default`

## Docker Compose

```bash
docker compose up --build
curl -s http://localhost:9101/metrics | head
```

## Сборка

```bash
make build
```

## Проверки качества

```bash
make test
make lint
```

Smoke-интеграция с локальным ClickHouse:

```bash
make preflight
docker compose up -d clickhouse
make integration-smoke
docker compose down -v
```

Полный локальный цикл:

```bash
make ci
```

Снять snapshot метрик и сохранить summary:

```bash
PROFILE=extended make metrics-snapshot
```

Сравнить с baseline exporter (должен быть доступен по `BASELINE_URL`):

```bash
BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare
```

## Документация

- [docks/requirements.md](docks/requirements.md) — требования
- [docs/metrics_matrix.md](docs/metrics_matrix.md) — матрица метрик и baseline
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — архитектура
- [docs/experiments.md](docs/experiments.md) — эксперименты
- [docs/experiments_results_template.md](docs/experiments_results_template.md) — шаблон таблиц/выводов для ВКР
- [docs/experiments_results_current.md](docs/experiments_results_current.md) — текущий статус прогонов
- [docs/status.md](docs/status.md) — текущий статус и открытые задачи

## Права в ClickHouse

См. [docks/grants_example.sql](docks/grants_example.sql).
