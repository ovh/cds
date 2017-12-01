-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_group" (workflow_id INT, group_id INT, role INT, PRIMARY KEY(group_id, workflow_id));

-- +migrate Down
DROP TABLE "workflow_group";