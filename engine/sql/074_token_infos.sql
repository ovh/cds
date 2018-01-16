-- +migrate Up
ALTER TABLE token ADD COLUMN description TEXT;
ALTER TABLE token ADD COLUMN creator TEXT;
ALTER TABLE token ADD COLUMN id BIGSERIAL PRIMARY KEY;
-- +migrate Down
ALTER TABLE token DROP COLUMN description;
ALTER TABLE token DROP COLUMN creator;
ALTER TABLE token DROP COLUMN id;
