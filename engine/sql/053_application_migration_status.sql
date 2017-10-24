-- +migrate Up
ALTER TABLE application ADD COLUMN workflow_migration VARCHAR(50) DEFAULT 'NOT_BEGUN';
ALTER TABLE project ADD COLUMN workflow_migration VARCHAR(50) DEFAULT 'NOT_BEGUN';

-- +migrate Down
ALTER table application DROP COLUMN workflow_migration;
ALTER table project DROP COLUMN workflow_migration;