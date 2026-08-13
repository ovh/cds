-- +migrate Up
ALTER TABLE authentified_user ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT false;

-- +migrate Down
ALTER TABLE authentified_user DROP COLUMN disabled;
