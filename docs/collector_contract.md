# Контракт добавления нового коллектора (модуля)

## Цель

Зафиксировать единый способ расширения exporter без правки orchestration-цикла.

## Интерфейс

Новый шаг реализует `CollectorStep`:

- `Name() string` — уникальный идентификатор шага (для registry и self-metrics);
- `MinProfile() config.Profile` — минимальный профиль (`safe/extended/aggressive`);
- `RequiredTables() []string` — системные таблицы ClickHouse, необходимые шагу;
- `RequiredColumns() []SchemaColumn` — опциональные колонки, без которых шаг должен быть отключён на этапе capability detection;
- `Collect(ctx, conn, sink) error` — сбор и публикация данных в `StepSink`.

## Где регистрировать шаг

Добавить шаг в `internal/collector/pipeline.go` в `buildStepRegistry()`.

Пример:

```go
collectorStep{
    name: "my_step",
    min: config.ProfileExtended,
    requiredTables: []string{"parts"},
    requiredColumns: []SchemaColumn{{Table: "parts", Column: "active"}},
    collector: collectMyStep,
}
```

## Где писать SQL

Рекомендуется добавить функцию вида `collect<MyStep>Step(...)` в `internal/collector/queries.go`.

Правила:

- использовать `ctx` (таймаут шага уже встроен через `TimeoutPolicy`);
- выполнять SQL через `QueryExecutor`, а не напрямую через ad-hoc helper;
- закрывать rows через `defer func() { _ = rows.Close() }()`;
- возвращать ошибку наверх (`StepErrorReporter` и fail-safe обрабатываются orchestration-слоем).

## Публикация метрик

Использовать только `StepSink`, а не прямую запись в поля Exporter.

Если нужен новый тип метрики, добавить метод в `StepSink` и реализацию в `sink.go`.

## Обязательные проверки перед merge

1. `make lint`
2. `go test ./...`
3. `make integration-smoke PROFILE=<нужный профиль>`

## Требования к метрикам

- Имя в пространстве `ch_exporter_*`;
- Тип соответствует природе данных:
  - counter для монотонных счётчиков;
  - gauge для моментальных значений;
  - histogram только для распределений;
- Контроль cardinality: при необходимости фильтры allow/deny и/или top-N.
