-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_run_secret" (
    "id" VARCHAR(36) NOT NULL PRIMARY KEY,
    "workflow_run_id" BIGINT NOT NULL,
    "context" TEXT NOT NULL,
    "type" VARCHAR(50) NOT NULL,
    "name" TEXT NOT NULL,
    "cypher_value" BYTEA,
    "sig" BYTEA,
    "signer" TEXT
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_SECRET', 'workflow_run_secret', 'workflow_run', 'workflow_run_id', 'id');

ALTER TABLE "workflow_run" ADD COLUMN read_only BOOLEAN;
UPDATE "workflow_run" set read_only = false;
ALTER TABLE "workflow_run" ALTER COLUMN read_only SET DEFAULT false;

UPDATE workflow_run SET read_only = true WHERE last_modified < NOW() - INTERVAL '1 month';

-- +migrate Down
DROP TABLE "workflow_run_secret";

ALTER TABLE "workflow_run" DROP COLUMN read_only;

