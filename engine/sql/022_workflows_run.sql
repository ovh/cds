-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_run" (
    id BIGSERIAL PRIMARY KEY,
    num BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    workflow_id BIGINT NOT NULL,
    workflow JSONB,
    start TIMESTAMP WITH TIME ZONE NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
    triggeredBy JSONB
);

SELECT create_foreign_key('FK_WORKFLOW_RUN_WORKFLOW', 'workflow_run', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_RUN_PROJECT', 'workflow_run', 'project', 'project_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_run" (
    workflow_run_id BIGINT NOT NULL,
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    num BIGINT NOT NULL,
    sub_num BIGINT NOT NULL,
    status TEXT NOT NULL,
    start TIMESTAMP WITH TIME ZONE NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
    done TIMESTAMP WITH TIME ZONE NOT NULL,
    hook_event JSONB,
    manual JSONB,
    source_node_runs JSONB,
    payload JSONB,
    pipeline_parameters JSONB,
    build_parameters JSONB,
    tests JSONB,
    commits JSONB,
    stages JSONB
);


SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_WORKFLOW_RUN', 'workflow_node_run', 'workflow_run', 'workflow_run_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_WORKFLOW_NODE', 'workflow_node_run', 'workflow_node', 'workflow_node_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_run_artifact" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_id BIGINT,
    name TEXT, 
    tag TEXT, 
    download_hash TEXT, 
    size BIGINT, 
    perm INT, 
    md5sum TEXT, 
    object_path TEXT, 
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_ARTIFACT_WORKFLOW_NODE_RUN', 'workflow_node_run_artifact', 'workflow_node_run', 'workflow_node_run_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_run_job" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_id BIGINT,
    variables JSONB,
    job JSONB,
    status TEXT,
    queued TIMESTAMP WITH TIME ZONE,
    start TIMESTAMP WITH TIME ZONE,
    done TIMESTAMP WITH TIME ZONE,
    model TEXT,
    spawninfos JSONB
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_JOB_WORKFLOW_NODE_RUN', 'workflow_node_run_job', 'workflow_node_run', 'workflow_node_run_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_run_job_logs" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_job_id BIGINT,
    workflow_node_run_id BIGINT,
    start TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE,
    done TIMESTAMP WITH TIME ZONE,
    step_order BIGINT,
    "value" BYTEA
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_JOGS_WORKFLOW_NODE_RUN', 'workflow_node_run_job_logs', 'workflow_node_run', 'workflow_node_run_id', 'id');

-- //TODO
SELECT create_index('workflow_run', 'IDX_WORKFLOW_LOAD_RUNS', 'project_id, workflow_id');

-- +migrate Down

DROP TABLE workflow_node_run_job_logs;
DROP TABLE workflow_node_run_artifact;
DROP TABLE workflow_node_run_job;
DROP TABLE workflow_node_run;
DROP TABLE workflow_run;