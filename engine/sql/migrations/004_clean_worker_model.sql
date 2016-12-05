-- +migrate Up

ALTER TABLE worker_model DROP COLUMN owner_id;

-- +migrate Down

ALTER TABLE worker_model ADD COLUMN owner_id BIGINT;
UPDATE worker_model SET owner_id = (SELECT id FROM "user" WHERE admin = true LIMIT 1);