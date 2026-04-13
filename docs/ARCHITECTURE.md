# Архитектура экспортёра

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
