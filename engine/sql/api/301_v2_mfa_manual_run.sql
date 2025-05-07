-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN admin_mfa boolean default false;
ALTER TABLE v2_workflow_run_job ADD COLUMN admin_mfa boolean default false;

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN admin_mfa;
ALTER TABLE v2_workflow_run_job DROP COLUMN admin_mfa;

