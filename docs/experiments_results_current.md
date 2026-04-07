# Текущий прогон экспериментов

## Статус

- Прогоны выполнены на локальном single-node стенде Docker.
- Smoke-прогон для профиля `extended`: успешно (`[smoke] ok`).
- Сняты snapshots для `safe`, `extended`, `aggressive`.
- Выполнено baseline-сравнение.

## Условия прогона

- Дата: 2026-04-06
- Стенд: Docker Desktop, `clickhouse/clickhouse-server:24.8`
- Доступ к CH: `default/clickhouse` (см. `docker-compose.yml`)
- Candidate exporter: текущий локальный build `./bin/ch-exporter`
- Baseline exporter: `f1yegor/clickhouse-exporter` (Docker Hub; образ GHCR недоступен без auth)

## Результаты (фактические)

| Вариант | Профиль | Unique metric names | Series count | p95 scrape duration (s) | CPU exporter (%) | RSS exporter (MB) | Комментарий |
|--------|---------|---------------------:|-------------:|-------------------------|-----------------:|------------------:|-------------|
| Новый экспортёр | safe | 40 | 1297 | n/a (snapshot, без окна PromQL) | n/a | n/a | `artifacts/experiments/exporter_safe_20260406_160347.summary.txt` |
| Новый экспортёр | extended | 47 | 1360 | n/a (snapshot, без окна PromQL) | 1.5 | 20.9 | CPU/RSS: `ps` после 30 scrape-запросов |
| Новый экспортёр | aggressive | 48 | 1381 | n/a (snapshot, без окна PromQL) | n/a | n/a | `artifacts/experiments/exporter_aggressive_20260406_160356.summary.txt` |
| Baseline | n/a | 1254 | 1276 | n/a (snapshot, без окна PromQL) | 2.37 | 18.4 | `f1yegor/clickhouse-exporter`, метрики из `baseline_compare_20260406_160622.md` и `docker stats` |

## Выводы по текущему прогону

1. **Покрытие candidate по числу series выше**, чем у baseline на этом стенде (`1360` vs `1276` в профильном сравнении `extended`).
2. **Unique names у baseline существенно выше** из-за его схемы именования (много индивидуальных имён метрик), поэтому сравнение по этому столбцу интерпретировать осторожно.
3. Замеры CPU/RSS показывают сопоставимый порядок ресурсов между candidate и baseline на коротком прогоне.
4. Для ВКР нужно добавить окно наблюдения (10+ минут) и посчитать p95 через PromQL (`histogram_quantile`) для более строгого сравнения.

## Артефакты

- Snapshot summaries:
  - `artifacts/experiments/exporter_safe_20260406_160347.summary.txt`
  - `artifacts/experiments/exporter_extended_20260406_160351.summary.txt`
  - `artifacts/experiments/exporter_aggressive_20260406_160356.summary.txt`
- Baseline compare report:
  - `artifacts/experiments/baseline_compare_20260406_160622.md`
