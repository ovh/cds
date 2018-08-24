-- +migrate Up
ALTER TABLE action_edge ADD COLUMN step_name TEXT DEFAULT '';

-- +migrate Down
ALTER TABLE action_edge DROP COLUMN step_name;
