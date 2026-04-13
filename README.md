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
- Готовые пресеты:
  - `examples/profiles/safe.yaml`
  - `examples/profiles/extended.yaml`
  - `examples/profiles/aggressive.yaml`
- Переменные окружения с префиксом `CH_EXPORTER_`: см. [internal/config/config.go](internal/config/config.go) (`LISTEN_ADDRESS`, `DSN`, `ADDRESS`, `PROFILE`, `PARTS_TOP_N`, `QUERY_TIMEOUT`, TLS, allowlist/denylist).
- Если часть `system.*` недоступна в конкретной версии ClickHouse, соответствующий шаг коллектора автоматически отключается (fail-safe), чтобы не генерировать ошибки на каждом scrape.
- Каждый шаг коллектора выполняется с ограничением `query_timeout` (per-step timeout).
- В `aggressive` действует жёсткий лимит `parts_top_n <= 100`.

Пример DSN: `clickhouse://default@localhost:9000/default`

Запуск с пресетом:

```bash
go run ./cmd/ch-exporter -config examples/profiles/extended.yaml
```

Запуск в одну команду через `make`:

```bash
make run-safe
make run-extended
make run-aggressive
```

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

## Права в ClickHouse

См. [docks/grants_example.sql](docks/grants_example.sql).
