-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_run" (
    id BIGSERIAL PRIMARY KEY,
    num BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    project_key TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    workflow JSONB,
    start TIMESTAMP WITH TIME ZONE NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL
);

SELECT create_foreign_key('FK_WORKFLOW_RUN_PROJECT', 'workflow_run', 'project', 'project_id', 'id');
SELECT create_index('workflow_run', 'IDX_WORKFLOW_RUN', 'project_key,workflow_name,id,num');

CREATE TABLE IF NOT EXISTS "workflow_node_run" (
    id BIGSERIAL PRIMARY KEY,
    workflow_run_id BIGINT NOT NULL,
    num BIGINT NOT NULL,
    sub_num BIGINT NOT NULL,
    start TIMESTAMP WITH TIME ZONE NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
    workflow_node_id BIGINT NOT NULL,
    pipeline_build_id BIGINT NOT NULL,
    hook_event JSONB,
    manual JSONB,
    trigger_id BIGINT,
    payload JSONB,
    pipeline_parameters JSONB
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_WORKFLOW_RUN', 'workflow_node_run', 'workflow_run', 'workflow_run_id', 'id');
SELECT create_index('workflow_node_run', 'IDX_WORKFLOW_NODE_RUN', 'workflow_run_id,id,num,sub_num');

-- +migrate Down

DROP TABLE workflow_node_run;
DROP TABLE workflow_run;