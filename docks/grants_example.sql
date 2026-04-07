-- Пример минимальных прав для учётной записи экспортёра (уточнить под версию CH и политику безопасности).
-- Чтение основных системных представлений:
GRANT SELECT ON system.metrics TO exporter_user;
GRANT SELECT ON system.events TO exporter_user;
GRANT SELECT ON system.asynchronous_metrics TO exporter_user;
GRANT SELECT ON system.replicas TO exporter_user;
GRANT SELECT ON system.merges TO exporter_user;
GRANT SELECT ON system.mutations TO exporter_user;
GRANT SELECT ON system.parts TO exporter_user;
GRANT SELECT ON system.disks TO exporter_user;
GRANT SELECT ON system.storage_policies TO exporter_user;
