-- +migrate Up

ALTER TABLE "w_node_context" DROP CONSTRAINT IF EXISTS "fk_w_node_context_integration";
SELECT create_foreign_key('FK_W_NODE_CONTEXT_INTEGRATION', 'w_node_context', 'project_integration', 'project_integration_id', 'id');

-- +migrate Down
SELECT 1;
