-- +migrate Up
SELECT create_unique_index('hatchery', 'IDX_HATCHERY_NAME', 'name');
-- +migrate Down
drop index IDX_HATCHERY_NAME;
