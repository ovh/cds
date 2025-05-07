-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN "concurrency" JSONB;
CREATE INDEX IDX_V2_WORKFLOW_RUN_CONCURRENCY ON v2_workflow_run(project_key, (concurrency->>'name'), (concurrency->>'scope'));

-- +migrate Down
DROP INDEX IDX_V2_WORKFLOW_RUN_CONCURRENCY;
ALTER TABLE v2_workflow_run DROP COLUMN "concurrency";