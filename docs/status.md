# Текущий статус проекта

## Выполнено

- Реализован внешний exporter на Go с endpoint'ами `/metrics`, `/healthz`, `/readyz`.
- Добавлены профили сбора: `safe`, `extended`, `aggressive`.
- Реализованы базовые и расширенные коллекторы (`system.metrics`, `system.events`, `system.asynchronous_metrics`, репликация, merges/mutations, диски, parts summary/top-N).
- Зафиксированы требования и ограничения: [docks/requirements.md](../docks/requirements.md).
- Подготовлены документы для ВКР: архитектура, матрица метрик, методика экспериментов.
- Добавлен CI с линтером и smoke-интеграцией: [.github/workflows/ci.yml](../.github/workflows/ci.yml).
- Добавлены вспомогательные скрипты для воспроизводимых экспериментов:
  - [scripts/preflight.sh](../scripts/preflight.sh)
  - [scripts/integration_smoke.sh](../scripts/integration_smoke.sh)
  - [scripts/collect_metrics_snapshot.sh](../scripts/collect_metrics_snapshot.sh)
  - [scripts/baseline_compare.sh](../scripts/baseline_compare.sh)

## Открытые задачи (остаток до финала ВКР)

- Зафиксировать p95 scrape duration через PromQL на окне 10+ минут (в текущем прогоне только snapshot).
- При необходимости добавить отдельный кластерный стенд (replication/shards) и явно выделить, что подтверждено экспериментом.
- Перенести итоговые цифры в основной текст ВКР (раздел «Эксперименты и сравнение»).

## Прогон в текущей среде

- Docker daemon доступен, smoke и snapshot-прогоны выполнены.
- Текущие числовые результаты и ограничения: [docs/experiments_results_current.md](./experiments_results_current.md).

## Как воспроизвести проверки

```bash
make test
make lint
make preflight
docker compose up -d clickhouse
make integration-smoke
PROFILE=extended make metrics-snapshot
BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare
docker compose down -v
```

