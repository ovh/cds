-- +migrate Up

CREATE TABLE IF NOT EXISTS "authentified_user" (
  id VARCHAR(36) PRIMARY KEY,
  username TEXT NOT NULL,
  fullname TEXT NOT NULL,
  ring VARCHAR(25) NOT NULL,
  date_creation TIMESTAMP WITH TIME ZONE,
  sig BYTEA
);


CREATE TABLE IF NOT EXISTS "authentified_user_migration" (
    authentified_user_id VARCHAR(36),
    user_id BIGINT,
    PRIMARY KEY (authentified_user_id, user_id)
);

SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_MIGRATION_USER', 'authentified_user_migration', 'user', 'user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_MIGRATION_AUTHENTIFIED_USER', 'authentified_user_migration', 'authentified_user', 'authentified_user_id', 'id');

CREATE TABLE IF NOT EXISTS "user_local_authentication" (
  user_id VARCHAR(36) PRIMARY KEY,
  encrypted_password BYTEA,
  verified BOOLEAN NOT NULL DEFAULT FALSE,
  sig BYTEA
);

SELECT create_foreign_key_idx_cascade('FK_LOCAL_AUTH_AUTHENTIFIED', 'user_local_authentication', 'authentified_user', 'user_id', 'id');


CREATE TABLE IF NOT EXISTS "user_contact" (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(36),
  type TEXT NOT NULL,
  value TEXT NOT NULL,
  primary_contact BOOLEAN NOT NULL DEFAULT FALSE,
  sig BYTEA
);

SELECT create_foreign_key_idx_cascade('FK_USER_CONTACT_AUTHENTIFIED', 'user_contact', 'authentified_user', 'user_id', 'id');

DROP TABLE IF EXISTS access_token_group;
DROP TABLE IF EXISTS access_token;

CREATE TABLE access_token
(
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(25),
    origin  VARCHAR(25),
    scopes JSONB
);

CREATE TABLE access_token_group
(
    access_token_id VARCHAR(255),
    group_id BIGINT,
    PRIMARY KEY (access_token_id, group_id)
);


SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_USER', 'access_token', 'authentified_user', 'user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_GROUP_ACCESS_TOKEN', 'access_token_group', 'access_token', 'access_token_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_GROUP_GROUP', 'access_token_group', 'group', 'group_id', 'id');
SELECT create_unique_index('access_token', 'IDX_ACCESS_TOKEN', 'user_id,name');

ALTER TABLE services ADD COLUMN IF NOT EXISTS token_id VARCHAR(64);
ALTER TABLE services ADD COLUMN IF NOT EXISTS maintainer JSONB;
ALTER TABLE services ADD COLUMN IF NOT EXISTS public_key BYTEA;
ALTER TABLE services ADD COLUMN IF NOT EXISTS encrypted_jwt BYTEA;
ALTER TABLE services ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE services ALTER COLUMN hash DROP NOT NULL;
SELECT create_unique_index('services', 'IDX_SERVICES_TOKENID', 'token_id');

ALTER TABLE worker RENAME TO old_worker;

CREATE TABLE worker
(
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    last_beat TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(255) NOT NULL,
    model_id BIGINT NOT NULL,
    job_run_id BIGINT,
    hatchery_id BIGINT NOT NULL,
    access_token_id VARCHAR(64) REFERENCES access_token(id)
);

SELECT create_foreign_key('FK_WORKER_MODEL', 'worker', 'worker_model', 'model_id', 'id');
select create_index('worker','IDX_WORKER_MODEL', 'model_id,id');
SELECT create_foreign_key('FK_WORKER_WORKFLOW_NODE_RUN_JOB', 'worker', 'workflow_node_run_job', 'job_run_id', 'id');
SELECT create_unique_index('worker', 'IDX_WORKER_JOB_RUN', 'job_run_id');
SELECT create_unique_index('worker', 'IDX_WORKER_NAME', 'name');
SELECT create_foreign_key('FK_WORKER_SERVICES', 'worker', 'services', 'hatchery_id', 'id');

-- TODO DELETE CASCASDE access_token when worker is removed

-- +migrate Down

DROP TABLE "access_token_group";
DROP TABLE "access_token";
DROP TABLE "authentified_user_migration";
DROP TABLE "user_local_authentication";
DROP TABLE "user_contact";
DROP TABLE "authentified_user";
DROP TABLE "worker";
ALTER TABLE old_worker RENAME TO worker;
