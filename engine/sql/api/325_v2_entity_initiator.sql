-- +migrate Up
ALTER TABLE entity ADD COLUMN initiator JSONB NULL;

-- +migrate Down
ALTER TABLE entity DROP COLUMN initiator;
