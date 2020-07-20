-- +migrate Up
ALTER TABLE pipeline_scheduler ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';

-- +migrate Down
ALTER table pipeline_scheduler DROP COLUMN timezone;