# Архитектура экспортёра

## Схема (PlantUML)

```plantuml
@startuml
title ClickHouse Prometheus Exporter

actor Prometheus
node "Exporter (Go)" as Exporter {
  component "HTTP Server\n/metrics /healthz /readyz" as Http
  component "Scrape Orchestrator" as Orchestrator
  component "Collector: system.metrics" as C1
  component "Collector: system.events" as C2
  component "Collector: async metrics" as C3
  component "Collector: replicas/merges/mutations/disks/parts" as C4
  component "Fail-safe + Step State\ncollector_enabled/last_success/last_error" as State
  component "Cardinality Controls\nallowlist/denylist, parts_top_n<=100" as Guard
}
database "ClickHouse" as CH

Prometheus --> Http : scrape
Http --> Orchestrator
Orchestrator --> C1
Orchestrator --> C2
Orchestrator --> C3
Orchestrator --> C4 : extended/aggressive

C1 --> CH : SQL system.metrics
C2 --> CH : SQL system.events
C3 --> CH : SQL system.asynchronous_metrics
C4 --> CH : SQL system.*

Orchestrator --> State : per-step status
Orchestrator --> Guard : filters/limits
State --> Http : self metrics
Guard --> Http : bounded label set
@enduml
```

## Компоненты

- **HTTP-сервер:** `/metrics` (Prometheus), `/healthz` (liveness), `/readyz` (readiness после успешного ping CH).
- **Пул подключений:** один `clickhouse.Conn` на процесс; лимит открытых соединений из конфига.
- **Оркестратор сбора:** на каждый scrape запускаются включённые коллекторы с общим контекстом и таймаутом; ошибки коллектора учитываются, остальные продолжают работу.
- **Расширяемость через контракт шага:** `CollectorStep` + `Registry` шагов; pipeline собирается декларативно по профилю (`safe/extended/aggressive`) без правок цикла `Collect`.
- **Per-step timeout:** каждый шаг коллектора ограничен `query_timeout`, чтобы один тяжёлый запрос не съедал весь `collect_timeout`.
- **Fail-safe по версиям CH:** если шаг падает из-за отсутствующей `system.*` таблицы/колонки (`Unknown table`, `Unknown identifier` и т.п.), шаг автоматически отключается до рестарта процесса и перестаёт зашумлять логи/ошибки scrape.
- **Коллекторы (по профилю):**
  - `safe`: `system.metrics`, `system.events`, `system.asynchronous_metrics`
  - `extended`: + реплики, merges/mutations (агрегаты), диски, сводка `system.parts`, demo-шаг `system.one`
  - `aggressive`: + top-N таблиц по числу кусков (`system.parts`), лимит N из конфига

## Конфигурация

- Файл YAML и/или переменные окружения `CH_EXPORTER_*`.
- Поля: адрес CH, пользователь, пароль, TLS, `profile`, таймауты, `parts_top_n` для aggressive.
- Cardinality controls:
  - allowlist/denylist для `system.metrics`, `system.events`, `system.asynchronous_metrics`;
  - allowlist/denylist баз данных для `parts_top`;
  - hard limit `parts_top_n <= 100`.

## Имена метрик

- Префикс `ch_exporter_` для пространства имён проекта.
- Лейблы: только стабильные идентификаторы (имена метрик CH, диски, при aggressive — ограниченный набор database/table).
- Метрики состояния шагов:
  - `ch_exporter_collector_enabled{step}`
  - `ch_exporter_collector_last_success_unix{step}`
  - `ch_exporter_collector_last_error_unix{step}`
