-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "concurrency" JSONB;
ALTER TABLE v2_workflow_run_job ADD COLUMN "vcs_server"  VARCHAR(255);
ALTER TABLE v2_workflow_run_job ADD COLUMN "repository"  VARCHAR(255);

UPDATE v2_workflow_run_job SET vcs_server = v2_workflow_run.vcs_server FROM v2_workflow_run WHERE v2_workflow_run.ID = v2_workflow_run_job.workflow_run_id;
UPDATE v2_workflow_run_job SET repository = v2_workflow_run.repository FROM v2_workflow_run WHERE v2_workflow_run.ID = v2_workflow_run_job.workflow_run_id;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "concurrency";
ALTER TABLE v2_workflow_run_job DROP COLUMN "vcs_server";
ALTER TABLE v2_workflow_run_job DROP COLUMN "repository";
