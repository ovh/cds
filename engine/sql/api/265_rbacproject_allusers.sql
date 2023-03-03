-- +migrate Up
ALTER TABLE rbac_project RENAME COLUMN "all" TO "all_users";

-- +migrate Down
ALTER TABLE rbac_project RENAME COLUMN "all_users" TO "all";
