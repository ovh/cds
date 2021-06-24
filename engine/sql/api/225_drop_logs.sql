-- +migrate Up
DROP TABLE "workflow_node_run_job_logs";
DROP TABLE "requirement_service_logs";

-- +migrate Down
CREATE TABLE IF NOT EXISTS "workflow_node_run_job_logs" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_job_id BIGINT,
    workflow_node_run_id BIGINT,
    start TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE,
    done TIMESTAMP WITH TIME ZONE,
    step_order BIGINT,
    "value" BYTEA
);
CREATE TABLE IF NOT EXISTS "requirement_service_logs" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_job_id BIGINT,
    workflow_node_run_id BIGINT,
    requirement_service_name TEXT,
    start TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE,
    "value" BYTEA
);
