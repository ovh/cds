-- +migrate Up
ALTER TABLE worker_model ADD COLUMN model JSONB;
ALTER TABLE worker_model ADD COLUMN registered_os text;
ALTER TABLE worker_model ADD COLUMN registered_arch text;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN model;
ALTER TABLE worker_model DROP COLUMN registered_os;
ALTER TABLE worker_model DROP COLUMN registered_arch;
