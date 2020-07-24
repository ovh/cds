-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN triggers_run JSONB;
ALTER TABLE workflow_run ADD COLUMN join_triggers_run JSONB;

-- +migrate Down
ALTER TABLE workflow_node_run DROP COLUMN triggers_run;
ALTER TABLE workflow_run DROP COLUMN join_triggers_run;