-- +migrate Up
CREATE INDEX idx_v2_workflow_run_cds_workflow_template_vcs_server ON v2_workflow_run((contexts -> 'cds' ->> 'workflow_template_vcs_server'));
CREATE INDEX idx_v2_workflow_run_cds_workflow_template_repository ON v2_workflow_run((contexts -> 'cds' ->> 'workflow_template_repository'));
CREATE INDEX idx_v2_workflow_run_cds_workflow_template ON v2_workflow_run((contexts -> 'cds' ->> 'workflow_template'));

-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_cds_workflow_template_vcs_server;
DROP INDEX IF EXISTS idx_v2_workflow_run_cds_workflow_template_repository;
DROP INDEX IF EXISTS idx_v2_workflow_run_cds_workflow_template;