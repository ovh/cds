-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_run_secret" (
    "workflow_run_id" BIGINT NOT NULL,
    "context" VARCHAR(25) NOT NULL,
    "name" TEXT NOT NULL,
    "cypher_value" BYTEA,
    "sig" BYTEA,
    "signer" TEXT,
    PRIMARY KEY(workflow_run_id, name)
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_SECRET', 'workflow_run_secret', 'workflow_run', 'workflow_run_id', 'id');

-- +migrate Down
DROP TABLE "workflow_run_secret";

