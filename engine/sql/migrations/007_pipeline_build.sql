-- +migrate Up
DROP table pipeline_history;
DROP table action_build;

ALTER TABLE pipeline_build ADD COLUMN 'stages' JSONB;

CREATE TABLE IF NOT EXISTS "pipeline_build_job" (
    id BIGSERIAL PRIMARY KEY,
    pipeline_build_id BIGINT,
    parameters JSONB,
    job JSONB,
    status TEXT,
    queued
    start
    done
    model
)

-- +migrate Down
CREATE TABLE IF NOT EXISTS "pipeline_history" (pipeline_build_id BIGINT, pipeline_id INT, application_id INT, environment_id INT, build_number INT, version BIGINT, status TEXT, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, data json, manual_trigger BOOLEAN, triggered_by BIGINT, parent_pipeline_build_id BIGINT, vcs_changes_branch TEXT, vcs_changes_hash TEXT, vcs_changes_author TEXT, scheduled_trigger BOOLEAN default FALSE, PRIMARY KEY(pipeline_id, application_id, build_number, environment_id));
CREATE TABLE IF NOT EXISTS "action_build" (id BIGSERIAL PRIMARY KEY, pipeline_action_id INT, args TEXT, status TEXT, pipeline_build_id INT, queued TIMESTAMP WITH TIME ZONE, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, worker_model_name TEXT);

ALTER TABLE pipeline_build DROP COLUMN 'stages';

DROP TABLE pipeline_build_job;