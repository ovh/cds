-- +migrate Up
CREATE TABLE IF NOT EXISTS "user_gpg_key" (
    "id" uuid PRIMARY KEY,
    "authentified_user_id" VARCHAR(36) NOT NULL,
    "key_id" VARCHAR(255) NOT NULL,
    "public_key" TEXT NOT NULL,
    "created" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('user_gpg_key', 'idx_unq_key_id', 'key_id');
SELECT create_foreign_key_idx_cascade('fk_gpg_key_user', 'user_gpg_key', 'authentified_user', 'authentified_user_id', 'id');

ALTER TABLE project_repository ADD COLUMN auth BYTEA;
ALTER TABLE project_repository ADD COLUMN clone_url TEXT;

CREATE TABLE IF NOT EXISTS "project_repository_analyze" (
    "id" uuid PRIMARY KEY,
    "created" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "last_modified" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "project_repository_id" uuid NOT NULL,
    "vcs_project_id" uuid NOT NULL,
    "project_key" VARCHAR(255),
    "branch" VARCHAR(255),
    "commit" VARCHAR(255),
    "status" VARCHAR(255) NOT NULL,
    "data" JSONB NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_foreign_key_idx_cascade('fk_project_repository_analyze_repository', 'project_repository_analyze', 'project_repository', 'project_repository_id', 'id');

-- +migrate Down
DROP TABLE user_gpg_key;
ALTER TABLE project_repository DROP COLUMN auth;
ALTER TABLE project_repository DROP COLUMN clone_url;
DROP TABLE project_repository_analyze;
