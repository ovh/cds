-- +migrate Up
ALTER TABLE project_notification ALTER COLUMN id TYPE VARCHAR(36);


-- +migrate Down
SELECT 1;