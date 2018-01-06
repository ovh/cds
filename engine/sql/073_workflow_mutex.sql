-- +migrate Up
ALTER TABLE workflow_node_context ADD COLUMN mutex BOOLEAN DEFAULT FALSE;
UPDATE workflow_node_context SET mutex = 'false';

ALTER TABLE workflow_node_run ADD COLUMN workflow_node_name text;
UPDATE workflow_node_run SET workflow_node_name = '';

-- +migrate Down
ALTER TABLE workflow_node_context DROP COLUMN mutex;
ALTER TABLE workflow_node_run DROP COLUMN workflow_node_run;

