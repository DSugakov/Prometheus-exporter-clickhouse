# Стратегия тестирования

## Цель

Подтвердить, что exporter:
- корректно собирает метрики ClickHouse;
- устойчив к ошибкам и несовместимостям схемы (`system.*`);
- контролирует нагрузку (таймауты, профили, cardinality);
- воспроизводимо запускается локально и в CI.

## Уровни тестирования

### 1) Статический анализ и unit-тесты

```bash
make lint
go test ./...
```

Критерий: без ошибок линтера и падений тестов.

Дополнительно (unit):
- проверка выбора шагов по профилям (`safe/extended/aggressive`);
- graceful-degradation: шаг отключается при unsupported-schema ошибке;
- стабильность registry (фиксированный порядок/состав шагов).

### 2) Smoke-интеграция

```bash
make preflight
docker compose up -d clickhouse
make integration-smoke PROFILE=extended
docker compose down -v
```

Проверки:
- `/readyz` отвечает `200`;
- в `/metrics` есть `ch_exporter_up`, `ch_exporter_system_metric_value`, `ch_exporter_scrape_step_duration_seconds`.

### 3) Профильные прогоны (`safe/extended/aggressive`)

```bash
docker compose up -d clickhouse
for p in safe extended aggressive; do
  CH_EXPORTER_ADDRESS=127.0.0.1:9000
  CH_EXPORTER_USERNAME=default
  CH_EXPORTER_PASSWORD=clickhouse
  CH_EXPORTER_PROFILE=$p
  CH_EXPORTER_LISTEN_ADDRESS=:9101
  ./bin/ch-exporter >/tmp/ch-exporter-$p.log 2>&1 &
  pid=$!
  sleep 4
  PROFILE=$p make metrics-snapshot
  kill $pid >/dev/null 2>&1 || true
  wait $pid 2>/dev/null || true
done
docker compose down -v
```

Критерии:
- артефакты созданы в `artifacts/experiments`;
- `extended` не меньше `safe` по series;
- `aggressive` не меньше `extended` по series.

### 4) Сравнение с baseline

```bash
docker run --rm -d --name ch-baseline -p 9116:9116 f1yegor/clickhouse-exporter \
  -scrape_uri http://default:clickhouse@host.docker.internal:8123/
BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare
docker rm -f ch-baseline
```

Критерий: отчёт `artifacts/experiments/baseline_compare_*.md` сформирован.

## Негативные сценарии (обязательно)

1. Неверный пароль к CH:
- ожидание: `/readyz` -> 503, `ch_exporter_up=0`, рост `scrape_errors_total`.

2. Неподдерживаемая схема:
- ожидание: проблемный шаг отключается (`ch_exporter_collector_enabled{step}=0`), процесс не падает.

3. Малый `query_timeout`:
- ожидание: timeout у отдельного шага, exporter продолжает отвечать `/metrics`.

## Проверка cardinality-фильтров

1. `system_event_allowlist`:
- в `ch_exporter_system_event_value` остаются только указанные события.

2. `system_metric_denylist`:
- запрещённые `metric` отсутствуют.

3. `parts_database_denylist`:
- нет `ch_exporter_table_active_parts{database="system",...}`.

4. `parts_top_n > 100`:
- exporter не стартует (ошибка валидации конфига).

## Минимальный regression-чек перед релизом

```bash
make lint
go test ./...
docker compose up -d clickhouse
make integration-smoke PROFILE=extended
PROFILE=extended make metrics-snapshot
BASELINE_URL=http://127.0.0.1:9116/metrics make baseline-compare
docker compose down -v
```

Сохранить артефакты:
- `artifacts/experiments/*.summary.txt`
- `artifacts/experiments/baseline_compare_*.md`
