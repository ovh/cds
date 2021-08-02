-- +migrate Up
CREATE INDEX IDX_ITEM_RUN_JOB ON item((api_ref->>'node_run_job_id'));

-- +migrate Down
DROP INDEX IDX_ITEM_RUN_JOB;
CREATE INDEX IDX_LOG_ITEM ON item(type, (api_ref->>'job_id'), (api_ref->>'step_order'));
