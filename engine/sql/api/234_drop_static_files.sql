-- +migrate Up
TRUNCATE workflow_node_run_static_files;
DROP TABLE workflow_node_run_static_files;

DELETE from action where name = 'Serve Static Files' and type = 'Builtin';

-- +migrate Down
CREATE TABLE workflow_node_run_static_files
(
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    entrypoint TEXT,
    public_url TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);

SELECT create_unique_index('workflow_node_run_static_files', 'IDX_NODE_RUN_STATIC_FILES_NAME_UNIQ', 'workflow_node_run_id,name');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_STATIC_FILES_NODE_RUN', 'workflow_node_run_static_files', 'workflow_node_run', 'workflow_node_run_id', 'id');
