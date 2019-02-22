-- +migrate Up
ALTER TABLE pipeline DROP COLUMN "type";

-- +migrate Down
ALTER TABLE pipeline ADD COLUMN "type" TEXT NOT NULL DEFAULT '';