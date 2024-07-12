-- +migrate Up
CREATE TABLE rbac_variableset
(
  "id"                BIGSERIAL PRIMARY KEY,
  "rbac_id"           uuid         NOT NULL,
  "all_users"         BOOLEAN      NOT NULL DEFAULT FALSE,
  "role"              VARCHAR(255) NOT NULL,
  "project_key"       VARCHAR(255) NOT NULL,
  "variablesets"      JSONB,
  "all_variablesets"  BOOLEAN NOT NULL DEFAULT FALSE,
  "sig"               BYTEA,
  "signer"            TEXT

);
CREATE INDEX rbac_variableset_name_gin_idx ON rbac_variableset USING gin (variablesets);
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset', 'rbac_variableset', 'rbac', 'rbac_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset_project', 'rbac_variableset', 'project', 'project_key', 'projectkey');
SELECT create_index('rbac_variableset', 'idx_rbac_variableset_project_role', 'role');

CREATE TABLE rbac_variableset_users
(
  "id"              BIGSERIAL PRIMARY KEY,
  "rbac_variableset_id" BIGINT,
  "user_id"         character varying(36),
  sig               BYTEA,
  signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset_users', 'rbac_variableset_users', 'rbac_variableset', 'rbac_variableset_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset_users_id', 'rbac_variableset_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_variableset_users', 'idx_unq_rbac_variableset_users', 'rbac_variableset_id,user_id');

CREATE TABLE rbac_variableset_groups
(
  "id"              BIGSERIAL PRIMARY KEY,
  "rbac_variableset_id" BIGINT,
  "group_id"        BIGINT,
  sig               BYTEA,
  signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset_groups', 'rbac_variableset_groups', 'rbac_variableset', 'rbac_variableset_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_variableset_groups_ids', 'rbac_variableset_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_variableset_groups', 'idx_unq_rbac_variableset_groups', 'rbac_variableset_id,group_id');


-- +migrate Down
DROP TABLE rbac_variableset_groups;
DROP TABLE rbac_variableset_users;
DROP TABLE rbac_variableset;

