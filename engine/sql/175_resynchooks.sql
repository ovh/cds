-- +migrate Up
WITH model AS (SELECT id::text from workflow_hook_model WHERE name = 'RepositoryWebHook'),
newhooks AS (
    SELECT t.hook, t.hook->'config'->'webHookID' as val, t.hook->>'uuid' as uuid, t.rn-1 as card, b.workflowID as workflowID
    FROM (
        SELECT w.workflow_data->'node'->'hooks' as hooks, w.id as workflowID
        FROM workflow w
        WHERE jsonb_typeof(w.workflow_data->'node'->'hooks') <> 'null'
    ) b, model m
    LEFT JOIN LATERAL jsonb_array_elements(b.hooks) WITH ORDINALITY AS t(hook, rn) ON TRUE
    WHERE t.hook->>'hook_model_id'::text = m.id AND
    ( t.hook->'config'->'webHookID'->>'value' IS NULL OR t.hook->'config'->'webHookID'->>'value' = '' )
),
hookUpdateData AS (
    SELECT
        workflow_node_hook.uuid,
        workflow_node_hook.config->'webHookID' as oldValue,
        newhooks.val as currentValue,
        newhooks.workflowID as workflowID,
        ('{node,hooks,' || newhooks.card || ',config,webHookID}')::text[] as path
    FROM workflow_node_hook, newhooks
    WHERE workflow_node_hook.uuid = newhooks.uuid
)
update workflow
set workflow_data = jsonb_set(workflow_data, hookUpdateData.path, hookUpdateData.oldValue::jsonb, true)
from hookUpdateData
where workflow.id = hookUpdateData.workflowID;

-- +migrate Down
SELECT 1;
