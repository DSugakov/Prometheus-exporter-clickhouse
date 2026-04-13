# Архитектура экспортёра

## C4: System Context (C1)

```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Context.puml

title C1 — ClickHouse Exporter System Context

Person(operator, "Operator/DevOps", "Настраивает и поддерживает мониторинг")
System(prometheus, "Prometheus", "Скрапит метрики и хранит time-series")
SystemDb(clickhouse, "ClickHouse", "Сервер БД, источник system.* метрик")
System(exporter, "Prometheus-exporter-clickhouse", "Внешний экспортёр метрик ClickHouse")
System_Ext(grafana, "Grafana/Alertmanager", "Визуализация и алертинг")

Rel(operator, exporter, "Конфигурирует и запускает")
Rel(prometheus, exporter, "Scrape /metrics", "HTTP")
Rel(exporter, clickhouse, "Читает system.*", "Native TCP (или DSN/TLS)")
Rel(grafana, prometheus, "Запрашивает и визуализирует")
Rel(operator, grafana, "Смотрит дашборды и алерты")

@enduml
```

## C4: Container (C2)

```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml

title C2 — ClickHouse Exporter Containers

Person(operator, "Operator/DevOps")
System_Boundary(boundary, "ClickHouse Monitoring Solution") {
  Container(prometheus, "Prometheus", "Go", "Скрапит /metrics и хранит time-series")
  Container(exporter, "Prometheus-exporter-clickhouse", "Go", "Собирает system.* метрики по шагам registry")
  ContainerDb(clickhouse, "ClickHouse", "DBMS", "Источник operational метрик")
  Container(grafana, "Grafana/Alertmanager", "Grafana/Alertmanager", "Дашборды и оповещения")
}

Rel(operator, exporter, "Настраивает профили/фильтры")
Rel(prometheus, exporter, "Scrape /metrics", "HTTP")
Rel(exporter, clickhouse, "SQL to system.*", "Native TCP/TLS")
Rel(grafana, prometheus, "Читает метрики")
Rel(operator, grafana, "Наблюдает и реагирует")

@enduml
```

## Компоненты

- **HTTP-сервер:** `/metrics` (Prometheus), `/healthz` (liveness), `/readyz` (readiness после успешного ping CH).
- **Пул подключений:** один `clickhouse.Conn` на процесс; лимит открытых соединений из конфига.
- **Оркестратор сбора:** pipeline шагов `CollectorStep`, собираемый декларативно по профилю.
- **Schema capability detection:** перед запуском шага проверяются требуемые таблицы (`RequiredTables`) и колонки (`RequiredColumns`) через `system.tables` и `system.columns`.
- **Per-step timeout policy:** шаги выполняются под `TimeoutPolicy`, чтобы один тяжелый запрос не съедал общий scrape budget.
- **Service layer:** SQL выполняется через `QueryExecutor`, ошибки и статусы шагов централизованы в `StepErrorReporter`.
- **Step observability metrics:** `ch_exporter_collector_enabled`, `ch_exporter_collector_last_success_unix`, `ch_exporter_collector_last_error_unix`.
- **Коллекторы (по профилю):**
  - `safe`: `system.metrics`, `system.events`, `system.asynchronous_metrics`
  - `extended`: + `replicas`, `merges`, `mutations`, `disks`, `parts_summary`
  - `aggressive`: + `parts_top` (top-N таблиц по числу кусков)

## Конфигурация

- Файл YAML и/или переменные окружения `CH_EXPORTER_*`.
- Поля: адрес CH, пользователь, пароль, TLS, `profile`, таймауты, `parts_top_n` для aggressive.

## Имена метрик

- Префикс `ch_exporter_` для пространства имён проекта.
- Лейблы: только стабильные идентификаторы (имена метрик CH, диски, при aggressive — ограниченный набор database/table).
- `system.events` публикуется как counter-семантика (`ch_exporter_system_event_total`) через delta-нормализацию.
