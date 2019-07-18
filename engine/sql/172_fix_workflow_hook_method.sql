-- +migrate Up

-- Add missing hook method in workflow data
WITH
  hookInfo AS (
    SELECT id AS workflow_id, json_array_elements((workflow_data->'node'->>'hooks')::JSON) AS hook FROM workflow
  ),
  hookInfoWithIndex AS (
    SELECT workflow_id, hook, row_number() OVER(PARTITION BY workflow_id) AS hook_index FROM hookInfo
  ),
  hookInfoFiltered AS (
    SELECT workflow_id, hook_index
    FROM hookInfoWithIndex
    WHERE hook->>'hook_model_name' = 'RepositoryWebHook' AND (hook->'config'->>'method' IS NULL OR hook->'config'->'method'->>'value' != 'POST')
  ),
  hookInfoWithPath AS (
    SELECT workflow_id, (concat('{node,hooks,', hook_index-1, ',config,method}'))::TEXT[] AS hook_path FROM hookInfoFiltered
  )
UPDATE workflow SET workflow_data = (
  jsonb_set(
    workflow_data,
    hookInfoWithPath.hook_path,
    '{"type":"string","value":"POST","configurable":false,"multiple_choice_list":null}',
    true
  )
)
FROM hookInfoWithPath WHERE workflow.id = hookInfoWithPath.workflow_id;

-- Add missing hook method in w_node_hook
WITH
  hookInfo AS (
    SELECT w_node_hook.id AS w_node_hook_id
    FROM w_node_hook
    JOIN workflow_hook_model ON workflow_hook_model.id = w_node_hook.hook_model_id
    WHERE workflow_hook_model.name = 'RepositoryWebHook'
    AND (w_node_hook.config->>'method' IS NULL OR w_node_hook.config->'method'->>'value' != 'POST')
  )
UPDATE w_node_hook SET config = (
  jsonb_set(
    config,
    '{method}',
    '{"type":"string","value":"POST","configurable":false,"multiple_choice_list":null}',
    true
  )
)
FROM hookInfo WHERE w_node_hook.id = hookInfo.w_node_hook_id;

-- +migrate Down
SELECT 1;
