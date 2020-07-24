-- +migrate Up
ALTER TABLE platform_model ADD COLUMN public BOOLEAN;
ALTER TABLE platform_model ADD COLUMN public_configurations JSONB;
update platform_model set public = false;

-- +migrate Down
ALTER TABLE platform_model DROP COLUMN public;
ALTER TABLE platform_model DROP COLUMN public_configurations;
