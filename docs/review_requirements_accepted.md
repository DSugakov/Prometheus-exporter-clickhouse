# Принятые правки по архитектурному ревью

Документ фиксирует принятые требования ревьюера и целевое состояние архитектуры.

## 1) Анализ текущей архитектуры

Текущая реализация построена как single-binary exporter:

- `cmd/ch-exporter/main.go` — старт, конфиг, HTTP (`/metrics`, `/healthz`, `/readyz`), registry.
- `internal/config/config.go` — YAML/env, профили, валидация.
- `internal/chclient/client.go` — ClickHouse client (native TCP/TLS).
- `internal/collector/*` — orchestration, шаги, SQL к `system.*`.

Поток данных:

`Config -> CH client -> Exporter.Collect() -> SQL system.* -> Prometheus metrics`.

## 2) Что сохраняем

- Простая структура пакетов.
- Профили нагрузки `safe/extended/aggressive`.
- Изоляция ошибок шагов (частичный сбой не валит весь scrape).
- Эксплуатационные endpoint'ы (`/healthz`, `/readyz`).
- Наличие архитектурной и экспериментальной документации.

## 3) Ключевые архитектурные требования (принято)

1. Явный расширяемый контракт коллекторов/шагов.
2. Registry/factory шагов и декларативная сборка pipeline по профилю.
3. Feature flags на уровне модулей (`allowlist/denylist`).
4. Capability detection по `system.*` (required tables, дальше optional columns).
5. Разделение таймаутов: общий scrape budget + per-step/per-query timeout.
6. Полная операционная конфигурация через YAML и env.
7. Контроль cardinality по коллекторам.
8. Unit-тесты модулей + integration smoke.
9. Стабильная семантика типов Prometheus-метрик.
10. Документированный контракт добавления нового коллектора.

## 4) Текущее состояние относительно требований

- Контракт шага + registry: **выполнено** (`CollectorStep`, `buildStepRegistry`, profile selection).
- Feature flags модулей: **выполнено** (`module_allowlist/module_denylist`).
- Capability detection по required tables + optional columns: **выполнено** (`RequiredTables` + `RequiredColumns`).
- Таймауты (collect + per-step query): **выполнено**.
- Общие сервисы (query execution, timeout policy, error reporting): **выполнено** (`QueryExecutor`, `TimeoutPolicy`, `StepErrorReporter`).
- Env/YAML операционные параметры: **выполнено**, включая `CH_EXPORTER_MAX_OPEN_CONNS`.
- Cardinality controls: **выполнено** (allow/deny + hard cap `parts_top_n`).
- Типы метрик: **выполнено**, `system.events` => `ch_exporter_system_event_total` (counter-semantics).
- Unit/smoke tests: **выполнено**, включая тесты delta/reset событий и registry-поведения.

Остаток:

- Критичных незакрытых требований ревьюера не осталось.

## 5) Рекомендуемый roadmap (A -> B -> C)

- **A (эволюционный):** декомпозиция монолита, тесты, таймауты, fail-safe.
- **B (целевая архитектура):** модульный registry и контракт шагов (текущее направление).
- **C (после стабилизации):** декларативные custom collectors с жёсткими guardrails.
