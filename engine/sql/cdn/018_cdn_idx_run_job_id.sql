-- +migrate Up
CREATE INDEX IDX_ITEM_RUN_JOB_ID ON item((api_ref->>'run_job_id'));

-- +migrate Down
DROP INDEX IDX_ITEM_RUN_JOB_ID;

