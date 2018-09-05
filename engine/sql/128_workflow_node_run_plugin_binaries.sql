-- +migrate Up
ALTER TABLE workflow_node_run_job DROP COLUMN platform_plugin_binaries;
ALTER TABLE workflow_node_run_job ADD COLUMN plugin_binaries JSONB;

-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN plugin_binaries;
ALTER TABLE workflow_node_run_job ADD COLUMN platform_plugin_binaries JSONB;
