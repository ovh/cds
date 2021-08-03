-- +migrate Up
CREATE INDEX IDX_ITEM_RUN_JOB ON item((api_ref->>'node_run_job_id'));

-- +migrate Down
DROP INDEX IDX_ITEM_RUN_JOB;
