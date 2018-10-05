-- +migrate Up

ALTER TABLE services ADD COLUMN config jsonb;

-- +migrate Down

ALTER TABLE services DROP COLUMN config;
