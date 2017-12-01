-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN to_delete BOOLEAN DEFAULT false;
ALTER TABLE workflow ADD COLUMN history_length BIGINT DEFAULT 20;
ALTER TABLE workflow ADD COLUMN purge_tags JSONB;
ALTER TABLE workflow_node_run DROP CONSTRAINT fk_workflow_node_run_workflow_run;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_WORKFLOW_RUN', 'workflow_node_run', 'workflow_run', 'workflow_run_id', 'id');
ALTER TABLE workflow_node_run_job_logs DROP CONSTRAINT fk_workflow_node_run_jogs_workflow_node_run;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_JOBS_WORKFLOW_NODE_RUN', 'workflow_node_run_job_logs', 'workflow_node_run', 'workflow_node_run_id', 'id');
ALTER TABLE workflow_node_run_job DROP CONSTRAINT fk_workflow_node_run_job_workflow_node_run;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_JOB_WORKFLOW_NODE_RUN', 'workflow_node_run_job', 'workflow_node_run', 'workflow_node_run_id', 'id');

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN to_delete;
ALTER TABLE workflow DROP COLUMN history_length;
ALTER TABLE workflow DROP COLUMN purge_tags;
ALTER TABLE workflow_node_run DROP CONSTRAINT fk_workflow_node_run_workflow_run;
SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_WORKFLOW_RUN', 'workflow_node_run', 'workflow_run', 'workflow_run_id', 'id');
ALTER TABLE workflow_node_run_job_logs DROP CONSTRAINT fk_workflow_node_run_jobs_workflow_node_run;
SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_JOGS_WORKFLOW_NODE_RUN', 'workflow_node_run_job_logs', 'workflow_node_run', 'workflow_node_run_id', 'id');
ALTER TABLE workflow_node_run_job DROP CONSTRAINT fk_workflow_node_run_job_workflow_node_run;
SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_JOB_WORKFLOW_NODE_RUN', 'workflow_node_run_job', 'workflow_node_run', 'workflow_node_run_id', 'id');
