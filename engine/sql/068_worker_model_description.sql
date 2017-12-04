-- +migrate Up
ALTER TABLE worker_model ADD COLUMN description TEXT NOT NULL default '';

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN description;
