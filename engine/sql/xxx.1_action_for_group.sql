-- +migrate Up

UPDATE action_parameter SET "description" = '' WHERE "description" IS NULL;
ALTER TABLE action_parameter ALTER COLUMN "description" SET NOT NULL;

-- +migrate Down

SELECT 1;
