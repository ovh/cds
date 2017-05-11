-- +migrate Up

ALTER TABLE workflow_node_context ADD COLUMN default_payload JSONB;
ALTER TABLE workflow_node_context ADD COLUMN default_pipeline_parameters JSONB;

-- +migrate Down

ALTER TABLE workflow_node_context DROP COLUMN default_payload;
ALTER TABLE workflow_node_context DROP COLUMN default_pipeline_parameters;