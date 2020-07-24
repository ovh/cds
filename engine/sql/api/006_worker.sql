-- +migrate Up
ALTER TABLE worker ADD CONSTRAINT "cst_worker_action_build_id" UNIQUE(action_build_id);

-- +migrate Down
ALTER TABLE worker DROP CONSTRAINT "cst_worker_action_build_id";