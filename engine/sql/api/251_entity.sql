-- +migrate Up
CREATE TABLE IF NOT EXISTS "entity" (
    "id" uuid PRIMARY KEY,
    "project_repository_id" uuid NOT NULL,
    "project_key" TEXT NOT NULL,
    "type" VARCHAR(255) NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "branch" VARCHAR(255) NOT NULL,
    "commit" VARCHAR(255) NOT NULL,
    "last_update" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "data" TEXT NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('entity', 'idx_unq_repo_branch_type_name', 'project_repository_id,branch,type,name');
SELECT create_foreign_key_idx_cascade('fk_entity_repository', 'entity', 'project_repository', 'project_repository_id', 'id');

-- +migrate Down
DROP TABLE entity;
