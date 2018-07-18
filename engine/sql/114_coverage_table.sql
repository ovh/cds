-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN vcs_server TEXT;
ALTER TABLE workflow_node_run ADD COLUMN workflow_id BIGINT;
ALTER TABLE workflow_node_run ADD COLUMN application_id BIGINT;

CREATE TABLE IF NOT EXISTS "workflow_node_run_coverage" (
    workflow_id BIGINT,
    workflow_run_id BIGINT,
    workflow_node_run_id BIGINT,
    application_id BIGINT,
    run_number BIGINT,
    repository VARCHAR(255),
    branch VARCHAR(255),
    report JSONB,
    trend JSONB,
    PRIMARY KEY (workflow_node_run_id)
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_COVERAGE_WORKFLOW_RUN', 'workflow_node_run_coverage', 'workflow_run', 'workflow_run_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_COVERAGE_APPLICATION', 'workflow_node_run_coverage', 'application', 'application_id', 'id');
SELECT create_index('workflow_node_run_coverage', 'IDX_WORKFLOW_NODE_RUN_COVERAGE_SEARCH', 'workflow_id,repository,branch,run_number,application_id');

-- +migrate Down
DROP TABLE "workflow_node_run_coverage";
ALTER TABLE workflow_node_run DROP COLUMN vcs_server;
ALTER TABLE workflow_node_run DROP COLUMN workflow_id;
