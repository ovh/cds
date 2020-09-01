-- +migrate Up

SELECT create_unique_index('service', 'IDX_SERVICE_NAME', 'name');

-- +migrate Down

DROP INDEX IF EXISTS "idx_service_name";
