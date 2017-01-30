-- +migrate Up
ALTER table pipeline_history RENAME TO pipeline_history_old;

ALTER TABLE build_log DROP CONSTRAINT fk_build_log_action_build;
ALTER TABLE worker DROP CONSTRAINT fk_worker_action_build;

DROP table action_build;

ALTER TABLE pipeline_build ADD COLUMN stages JSONB;

CREATE TABLE IF NOT EXISTS "pipeline_build_job" (
    id BIGSERIAL PRIMARY KEY,
    pipeline_build_id BIGINT,
    parameters JSONB,
    job JSONB,
    status TEXT,
    queued TIMESTAMP WITH TIME ZONE,
    start TIMESTAMP WITH TIME ZONE,
    done TIMESTAMP WITH TIME ZONE,
    model TEXT
);
select create_index('pipeline_build_job','IDX_PIPELINE_BUILD_JOB', 'pipeline_build_id');
select create_foreign_key('FK_PIPELINE_BUILD_JOB_PIPELINE_BUILD', 'pipeline_build_job', 'pipeline_build', 'pipeline_build_id', 'id');

ALTER TABLE build_log ADD COLUMN pipeline_build_id BIGINT;
select create_foreign_key('FK_BUILD_LOG_PIPELINE_BUILD', 'build_log', 'pipeline_build', 'pipeline_build_id', 'id');
select create_index('build_log','IDX_BUILD_LOG_PIPELINE_BUILD', 'pipeline_build_id');

-- +migrate Down
ALTER table pipeline_history_old RENAME TO pipeline_history;

CREATE TABLE IF NOT EXISTS "action_build" (id BIGSERIAL PRIMARY KEY, pipeline_action_id INT, args TEXT, status TEXT, pipeline_build_id INT, queued TIMESTAMP WITH TIME ZONE, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, worker_model_name TEXT);
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_BUILD_ID', 'pipeline_build_id');
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID', 'pipeline_action_id');
select create_unique_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID_BUILD_ID', 'pipeline_build_id,pipeline_action_id');
select create_foreign_key('FK_WORKER_ACTION_BUILD', 'worker', 'action_build', 'action_build_id', 'id');
select create_foreign_key('FK_BUILD_LOG_ACTION_BUILD', 'build_log', 'action_build', 'action_build_id', 'id');


ALTER TABLE pipeline_build DROP COLUMN stages;

DROP TABLE IF EXISTS pipeline_build_job;

ALTER TABLE build_log DROP COLUMN pipeline_build_id;