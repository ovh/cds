-- +migrate Up
ALTER TABLE v2_workflow_run_job DROP integrations;

-- +migrate Down
ALTER TABLE v2_workflow_run_job ADD integrations JSONB;
