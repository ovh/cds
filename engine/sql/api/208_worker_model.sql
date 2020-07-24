-- +migrate Up

ALTER TABLE "worker_model" DROP COLUMN IF EXISTS image;

-- +migrate Down

ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS image TEXT;
