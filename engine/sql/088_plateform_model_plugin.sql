-- +migrate Up
ALTER TABLE platform_model ADD COLUMN platform_model_plugin JSONB;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN platform_model_plugin;
