-- +migrate Up
ALTER TABLE integration_model ADD COLUMN event BOOLEAN NOT NULL DEFAULT false;

-- +migrate Down
ALTER TABLE integration_model DROP COLUMN event;
