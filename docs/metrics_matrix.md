# Матрица метрик и классификация

## Baseline для сравнения (эксперименты)

- [prometheus-community/clickhouse-exporter](https://github.com/prometheus-community/clickhouse-exporter) — эталон для сравнения покрытия.
- Встроенная отдача метрик ClickHouse (если включена на стенде) — опциональный второй baseline.

## Методика подсчёта «сколько метрик»

- **Имена метрик (unique):** число уникальных `__name__` в одном scrape при фиксированном профиле.
- **Временные ряды:** число серий с учётом лейблов (для отчёта о cardinality); сравнивать осторожно между профилями.

## Классификация по доменам

| Класс      | Назначение | Основные источники в CH |
|-----------|------------|-------------------------|
| Ресурсы   | CPU/память/диск/сеть (как видно из CH) | `system.asynchronous_metrics`, частично `system.metrics` |
| Запросы   | нагрузка, ошибки, задержки (агрегированно) | `system.events`, `system.metrics`; опционально агрегаты `system.query_log` (не в `safe`) |
| Таблицы/данные | партиции, merges, мутации | `system.parts` (агрегаты), `system.merges`, `system.mutations` |
| Кластер   | реплики, очереди | `system.replicas`, `system.replication_queue` (если есть) |

## Матрица (сводка)

| Имя (префикс экспортёра) | Класс | Источник | Профиль | Риск cardinality |
|--------------------------|-------|----------|---------|------------------|
| `ch_exporter_up` | ops | ping | all | низкий |
| `ch_exporter_system_metric_value` | ресурсы | `system.metrics` | safe+ | низкий (label: metric) |
| `ch_exporter_system_event_total` | запросы | `system.events` | safe+ | низкий (label: event) |
| `ch_exporter_async_metric_value` | ресурсы | `system.asynchronous_metrics` | safe+ | средний (metric name) |
| `ch_exporter_replicas_*` | кластер | `system.replicas` | extended+ | средний |
| `ch_exporter_merge_*` | таблицы | `system.merges` | extended+ | низкий (агрегаты) |
| `ch_exporter_mutation_*` | таблицы | `system.mutations` | extended+ | низкий (агрегаты) |
| `ch_exporter_disk_*` | ресурсы | `system.disks` | extended+ | низкий |
| `ch_exporter_parts_*` | таблицы | `system.parts` | extended+ | настраиваемый |

Подробные SQL и лейблы задаются реализацией коллекторов и конфигом.
