-- +migrate Up

CREATE TABLE "service_status" (
    id BIGSERIAL PRIMARY KEY,
    auth_session_id VARCHAR(36),
    service_id BIGSERIAL NOT NULL,
    monitoring_status JSONB
);

SELECT create_foreign_key_idx_cascade('FK_SERVICE_STATUS_AUTH_SESSION', 'service_status', 'auth_session', 'auth_session_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_SERVICE_STATUS_SERVICE', 'service_status', 'service', 'service_id', 'id');
SELECT create_unique_index('service_status', 'idx_service_status_uniq', 'auth_session_id,service_id');

-- +migrate Down

DROP table IF EXISTS "service_status";
