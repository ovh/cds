-- +migrate Up
CREATE TABLE rbac_workflow (
  "id"            BIGSERIAL PRIMARY KEY,
  "rbac_id"       uuid         NOT NULL,
  "all_users"     BOOLEAN      NOT NULL DEFAULT FALSE,
  "role"          VARCHAR(255) NOT NULL,
  "project_key"       VARCHAR(255) NOT NULL,
  "workflows"     JSONB,
  "all_workflows" BOOLEAN NOT NULL DEFAULT FALSE,
  "sig"           BYTEA,
  "signer"        TEXT

);
CREATE INDEX rbac_workflow_name_gin_idx ON rbac_workflow USING gin (workflows);
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow', 'rbac_workflow', 'rbac', 'rbac_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow_project', 'rbac_workflow', 'project_key', 'project', 'projectkey');
SELECT create_index('rbac_workflow', 'idx_rbac_workflow_project_role', 'role');

CREATE TABLE rbac_workflow_users
(
  "id"              BIGSERIAL PRIMARY KEY,
  "rbac_workflow_id" BIGINT,
  "user_id"         character varying(36),
  sig               BYTEA,
  signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow_users', 'rbac_workflow_users', 'rbac_workflow', 'rbac_workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow_users_id', 'rbac_workflow_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_workflow_users', 'idx_unq_rbac_workflow_users', 'rbac_workflow_id,user_id');

CREATE TABLE rbac_workflow_groups
(
  "id"              BIGSERIAL PRIMARY KEY,
  "rbac_workflow_id" BIGINT,
  "group_id"        BIGINT,
  sig               BYTEA,
  signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow_groups', 'rbac_workflow_groups', 'rbac_workflow', 'rbac_workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_workflow_groups_ids', 'rbac_workflow_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_workflow_groups', 'idx_unq_rbac_workflow_groups', 'rbac_workflow_id,group_id');


-- +migrate Down
DROP TABLE rbac_workflow_groups;
DROP TABLE rbac_workflow_users;
DROP TABLE rbac_workflow;

