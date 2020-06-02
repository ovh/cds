-- +migrate Up

ALTER TABLE "environment" ADD COLUMN IF NOT EXISTS created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
UPDATE "environment" SET created = last_modified WHERE created > last_modified;

-- +migrate Down

SELECT 1;
