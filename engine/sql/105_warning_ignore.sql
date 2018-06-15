-- +migrate Up
ALTER TABLE warning ADD COLUMN ignored boolean DEFAULT false;

-- +migrate Down
ALTER TABLE warning DROP COLUMN ignored;