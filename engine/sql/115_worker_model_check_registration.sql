-- +migrate Up
ALTER TABLE worker_model ADD COLUMN check_registration BOOLEAN DEFAULT true;
UPDATE worker_model SET check_registration = false;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN check_registration;
