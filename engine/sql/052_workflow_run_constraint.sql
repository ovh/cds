-- +migrate Up
ALTER TABLE workflow_run DROP CONSTRAINT fk_workflow_run_workflow;
DROP INDEX idx_fk_workflow_run_workflow;
SELECT create_foreign_key_idx_cascade('fk_workflow_run_workflow', 'workflow_run', 'workflow', 'workflow_id', 'id');

-- +migrate Down
ALTER TABLE workflow_run DROP CONSTRAINT fk_workflow_run_workflow;
SELECT create_foreign_key('fk_workflow_run_workflow', 'workflow_run', 'workflow', 'workflow_id', 'id');