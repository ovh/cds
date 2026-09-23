-- Measuring the log coverage of a window means listing the jobs that terminated inside it. Both run
-- versions are dated by a column nothing indexes today, so the listing is a scan of the two largest
-- tables of the instance, every few minutes.

-- +migrate Up notransaction

-- CONCURRENTLY, since both tables are written to constantly. It cannot go through create_index():
-- a function always runs inside a transaction, which the concurrent build refuses.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v2_workflow_run_job_ended
	ON v2_workflow_run_job (ended);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_node_run_done
	ON workflow_node_run (done);

-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_job_ended;
DROP INDEX IF EXISTS idx_workflow_node_run_done;
