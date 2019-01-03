-- +migrate Up
DROP TABLE IF EXISTS pipeline_build_job;
DROP TABLE IF EXISTS received_hook;
DROP TABLE IF EXISTS hook;
DROP TABLE IF EXISTS pipeline_scheduler_execution;
DROP TABLE IF EXISTS pipeline_scheduler;
DROP TABLE IF EXISTS poller_execution_old;
DROP TABLE IF EXISTS poller_execution;
DROP TABLE IF EXISTS poller;
DROP TABLE IF EXISTS application_pipeline_notif;

-- +migrate Down
SELECT 1;
