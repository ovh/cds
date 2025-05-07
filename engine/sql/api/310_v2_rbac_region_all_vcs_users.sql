-- +migrate Up
ALTER TABLE rbac_region ADD COLUMN all_vcs_users BOOLEAN DEFAULT false;
ALTER TABLE rbac_project ADD COLUMN all_vcs_users BOOLEAN DEFAULT false;

-- +migrate Down
ALTER TABLE rbac_region DROP COLUMN all_vcs_users;
ALTER TABLE rbac_project DROP COLUMN all_vcs_users;
