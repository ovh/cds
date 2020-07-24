-- +migrate Up
ALTER table application DROP COLUMN workflow_migration;
ALTER table project DROP COLUMN workflow_migration;

-- +migrate Down
ALTER TABLE application ADD COLUMN workflow_migration VARCHAR(50) DEFAULT 'DONE';
ALTER TABLE project ADD COLUMN workflow_migration VARCHAR(50) DEFAULT 'NOT_BEGUN';
