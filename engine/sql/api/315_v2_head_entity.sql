-- +migrate Up
ALTER TABLE entity ADD COLUMN head boolean default false;
UPDATE entity SET head = true where commit = 'HEAD';
ALTER TABLE v2_workflow_hook ADD COLUMN head boolean default false;
UPDATE v2_workflow_hook SET head = true where commit = 'HEAD';

-- +migrate Down
ALTER TABLE entity DROP COLUMN head;
ALTER TABLE v2_workflow_hook DROP COLUMN head;