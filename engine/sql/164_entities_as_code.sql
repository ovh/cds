-- +migrate Up
ALTER TABLE pipeline ADD COLUMN from_repository TEXT DEFAULT '';
ALTER TABLE application ADD COLUMN from_repository TEXT DEFAULT '';
ALTER TABLE environment ADD COLUMN from_repository TEXT DEFAULT '';

-- +migrate Down
ALTER TABLE pipeline DROP COLUMN from_repository;
ALTER TABLE application DROP COLUMN from_repository;
ALTER TABLE environment DROP COLUMN from_repository;

