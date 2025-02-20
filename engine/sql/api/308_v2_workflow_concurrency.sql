-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "concurrency" JSONB;
ALTER TABLE v2_workflow_run_job ADD COLUMN "vcs_server"  VARCHAR(255);
ALTER TABLE v2_workflow_run_job ADD COLUMN "repository"  VARCHAR(255);


-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "concurrency";
ALTER TABLE v2_workflow_run_job DROP COLUMN "vcs_server";
ALTER TABLE v2_workflow_run_job DROP COLUMN "repository";
