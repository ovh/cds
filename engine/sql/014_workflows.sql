-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_foreign_key_idx_cascade(fk_name text, table_name_child text, table_name_parent text, column_name_child text, column_name_parent text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from information_schema.table_constraints as tc
  where constraint_type = 'FOREIGN KEY'
    and tc.table_name = lower(table_name_child)
    and tc.constraint_name = lower(fk_name);

  if l_count = 0 then
     execute 'alter table "' || table_name_child || '" ADD CONSTRAINT ' || fk_name || ' FOREIGN KEY(' || column_name_child || ') REFERENCES "' || table_name_parent || '"(' || column_name_parent || ') ON DELETE CASCADE';   
     execute create_index(table_name_child, 'IDX_' || fk_name, column_name_child);
  end if; 
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TABLE IF NOT EXISTS "workflow" (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    root_node_id BIGINT
);

SELECT create_index('workflow', 'IDX_WORKFLOW_NAME', 'name');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_PROJECT', 'workflow', 'project', 'project_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node" (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    pipeline_id BIGINT NOT NULL
);

SELECT create_foreign_key('FK_WORKFLOW_ROOT_NODE', 'workflow', 'workflow_node', 'root_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_WORKFLOW', 'workflow_node', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_PIPELINE', 'workflow_node', 'pipeline', 'pipeline_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_trigger" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL,
    conditions JSONB
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_context" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    application_id BIGINT,
    environment_id BIGINT
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_CONTEXT_WORKFLOW_NODE', 'workflow_node_context', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_APPLICATION', 'workflow_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_ENVIRONMENT', 'workflow_node_context', 'environment', 'environment_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_hook_model" (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    image TEXT NOT NULL,
    command TEXT NOT NULL,
    default_config JSONB
);

CREATE TABLE IF NOT EXISTS "workflow_node_hook" (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    workflow_node_id BIGINT NOT NULL,
    workflow_hook_model_id BIGINT NOT NULL,
    conditions JSONB,
    config JSONB
);

SELECT create_index('workflow_node_hook', 'IDX_WORKFLOW_NODE_HOOK_UUID', 'uuid');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_HOOK_WORKFLOW_NODE', 'workflow_node_hook', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_HOOK_WORKFLOW_HOOK_MODEL', 'workflow_node_hook', 'workflow_hook_model', 'workflow_hook_model_id', 'id');

-- +migrate Down
DROP TABLE workflow_hook_model CASCADE;
DROP TABLE workflow_node_hook CASCADE;
DROP TABLE workflow_node_context CASCADE;
DROP TABLE workflow_node_trigger CASCADE;
DROP TABLE workflow_node CASCADE;
DROP TABLE workflow CASCADE;

