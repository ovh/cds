-- +migrate Up
ALTER TABLE project_concurrency ADD COLUMN "if" TEXT NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE project_concurrency DROP COLUMN "if";