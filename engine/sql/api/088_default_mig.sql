-- +migrate Up
ALTER TABLE application ALTER COLUMN workflow_migration SET DEFAULT 'DONE';

-- +migrate Down
ALTER TABLE application ALTER COLUMN workflow_migration SET DEFAULT 'NOT_BEGUN';
