-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN "initiator" JSONB;
ALTER TABLE v2_workflow_run_job ADD COLUMN "initiator" JSONB;
ALTER TABLE rbac_project ADD COLUMN "vcs_users" JSONB;
ALTER TABLE rbac_workflow ADD COLUMN "vcs_users" JSONB;
ALTER TABLE rbac_variableset ADD COLUMN "vcs_users" JSONB;
ALTER TABLE rbac_region ADD COLUMN "vcs_users" JSONB;
ALTER TABLE project_key ADD COLUMN "long_key_id" VARCHAR(16);
UPDATE project_key SET long_key_id = '';

ALTER TABLE entity DROP CONSTRAINT fk_entity_user;

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN "initiator";
ALTER TABLE v2_workflow_run_job DROP COLUMN "initiator";
ALTER TABLE rbac_project DROP COLUMN "vcs_users";
ALTER TABLE rbac_workflow DROP COLUMN "vcs_users";
ALTER TABLE rbac_variableset DROP COLUMN "vcs_users";
ALTER TABLE rbac_region DROP COLUMN "vcs_users";
ALTER TABLE project_key DROP COLUMN "long_key_id";

ALTER TABLE entity ADD CONSTRAINT fk_entity_user FOREIGN KEY (user_id) REFERENCES authentified_user (id) ON DELETE SET NULL;
