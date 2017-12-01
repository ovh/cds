-- +migrate Up
ALTER TABLE workflow_run_artifacts ADD COLUMN created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
ALTER TABLE workflow_run_artifacts DROP COLUMN workflow_id;
ALTER TABLE workflow_run_artifacts ADD COLUMN workflow_run_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_WORKFLOW', 'workflow_run_artifacts', 'workflow_run', 'workflow_run_id', 'id');

DROP TABLE workflow_node_run_artifact;

ALTER TABLE workflow_run_artifacts RENAME TO workflow_node_run_artifacts;

-- +migrate Down
ALTER TABLE workflow_run_artifacts DROP COLUMN created;
ALTER TABLE workflow_run_artifacts DROP COLUMN workflow_run_id;
ALTER TABLE workflow_run_artifacts ADD COLUMN workflow_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_WORKFLOW', 'workflow_run_artifacts', 'workflow_run', 'workflow_id', 'id');

DROP TABLE workflow_node_run_artifacts;

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
