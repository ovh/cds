-- +migrate Up
ALTER TABLE workflow ADD COLUMN generate_badge BOOLEAN DEFAULT false;

-- +migrate Down
ALTER TABLE workflow DROP COLUMN generate_badge;