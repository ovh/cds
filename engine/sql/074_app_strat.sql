-- +migrate Up
ALTER TABLE application ADD COLUMN vcs_strategy JSONB;
UPDATE action_parameter SET type = 'string' where type = 'key';

-- +migrate Down
ALTER TABLE application DROP COLUMN vcs_strategy;


