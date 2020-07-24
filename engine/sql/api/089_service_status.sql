-- +migrate Up
ALTER TABLE services ADD COLUMN monitoring_status JSONB;

-- +migrate Down
ALTER TABLE services DROP COLUMN monitoring_status;
