-- +migrate Up
CREATE TABLE IF NOT EXISTS "requirement_service_logs" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_job_id BIGINT,
    workflow_node_run_id BIGINT,
    requirement_service_name TEXT,
    start TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE,
    "value" BYTEA
);

SELECT create_foreign_key_idx_cascade('FK_REQUIREMENT_SERVICE_LOGS_WORKFLOW_NODE_RUN', 'requirement_service_logs', 'workflow_node_run', 'workflow_node_run_id', 'id');

-- +migrate Down
DROP TABLE requirement_service_logs;
