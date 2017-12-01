-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_run_tag" (
    workflow_run_id BIGINT,
    tag VARCHAR(256) NOT NULL,
    value VARCHAR(256) NOT NULL,
    PRIMARY KEY (workflow_run_id, tag)
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_TAG_WORKFLOW_RUN', 'workflow_run_tag', 'workflow_run', 'workflow_run_id', 'id');

-- +migrate Down
DROP TABLE workflow_run_tag;
