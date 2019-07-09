-- +migrate Up
WITH
  hookInfo AS (
    SELECT id AS workflow_id, json_array_elements((workflow_data->'node'->>'hooks')::JSON) AS hook FROM workflow
  ),
  hookInfoWithIndex AS (
    SELECT workflow_id, hook, row_number() OVER(PARTITION BY id) AS hook_index FROM hookInfo
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
    '{"type":"string","value":"POST","configurable":true,"multiple_choice_list":null}',
    true
  )
)
FROM hookInfoWithPath WHERE workflow.id = hookInfoWithPath.workflow_id;

-- +migrate Down
SELECT 1;
