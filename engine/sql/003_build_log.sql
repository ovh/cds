-- +migrate Up
ALTER TABLE build_log ADD COLUMN step_order int;

-- +migrate Down
ALTER table build_log DROP COLUMN step_order;