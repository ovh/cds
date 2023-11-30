-- +migrate Up
ALTER TABLE v2_workflow_run_result ADD COLUMN run_attempt BIGINT DEFAULT 1;

-- +migrate Down
ALTER TABLE v2_workflow_run_result DROP COLUMN run_attempt;
