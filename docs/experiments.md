# Эксперименты и сравнение

## Цель

Подтвердить три утверждения:

1. Экспортёр стабильно работает без server-side настройки ClickHouse.
2. Покрытие метрик больше baseline при контролируемой кардинальности.
3. Нагрузка на стенд (CPU/RAM/latency) приемлема для выбранного профиля.

## Baseline

- Основной baseline: [prometheus-community/clickhouse-exporter](https://github.com/prometheus-community/clickhouse-exporter).
- Опциональный baseline: встроенная отдача метрик CH (если доступна в конкретном стенде).

## Стенд

- По умолчанию: single-node ClickHouse через [docker-compose.yml](../docker-compose.yml).
- Профили экспортёра: `safe`, `extended`, `aggressive`.
- Для кластерных метрик допускается отдельный стенд; в отчёте обязательно разделять:
  - что проверено экспериментально;
  - что обосновано документацией.

## Методика измерений

### 1) Smoke/готовность

```bash
docker compose up -d clickhouse
go build -o ./bin/ch-exporter ./cmd/ch-exporter/
./scripts/integration_smoke.sh ./bin/ch-exporter
```

Критерий: проходят `/healthz`, `/readyz`, в `/metrics` есть ключевые метрики `ch_exporter_up`, `ch_exporter_system_metric_value`.

### 2) Подсчёт покрытия

- **Имена метрик (unique):** число уникальных `__name__` в scrape.
- **Число серий:** с учетом лейблов (для оценки cardinality).

Пример:

```bash
PROFILE=extended make metrics-snapshot
```

Артефакты сохраняются в `artifacts/experiments/`:

- `exporter_<profile>_<timestamp>.metrics`
- `exporter_<profile>_<timestamp>.summary.txt`

### 3) Нагрузка

- Измерить `CPU%` и `RSS` процесса экспортёра.
- Зафиксировать частоту scrape (например 15s) и длительность окна (например 10 минут).
- Для честного сравнения baseline и новый экспортёр прогоняются при одинаковых параметрах стенда.

### 4) Сравнение с baseline

Пример запуска community baseline в Docker:

```bash
docker run --rm -d --name ch-baseline -p 9116:9116 \
  -e CLICKHOUSE_URL=http://host.docker.internal:8123 \
  -e CLICKHOUSE_USER=default \
  -e CLICKHOUSE_PASSWORD= \
  ghcr.io/prometheus-community/clickhouse-exporter:latest
```

Подними baseline экспортёр (например community exporter) и сделай сравнение:

```bash
BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare
```

Скрипт создаст отчёт `artifacts/experiments/baseline_compare_<timestamp>.md`.

## Шаблон фиксации результатов

Использовать [docs/experiments_results_template.md](experiments_results_template.md).

Обязательные колонки:

- версия ClickHouse;
- профиль экспортёра;
- уникальные имена метрик;
- число серий;
- p95 scrape duration;
- CPU/RSS экспортёра;
- заметки по cardinality и ограничениям.

## Ограничения и риски интерпретации

- Single-node стенд не покрывает все кластерные сценарии.
- Значения производительности зависят от hardware и фоновой нагрузки.
- Сравнение по количеству серий корректно только при одинаковых правилах relabeling и scrape-конфигурации.
