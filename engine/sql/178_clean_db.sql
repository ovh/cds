-- +migrate Up
DELETE FROM workflow_node_run_job WHERE id IN (
    select j.id
    from workflow_node_run_job j
    JOIN workflow_node_run wnr ON wnr.id = j.workflow_node_run_id
    WHERE wnr.status = 'Stopped'
);

-- clean w_node_trigger child_node
WITH workflowInfo AS (
	SELECT id, name, CAST(workflow_data->'node'->>'id' AS BIGINT) as rootNodeID
	FROM workflow
),
oldNode as (
	SELECT w_node.id as nodeID, w_node.name as nodeName, workflowInfo.id as wID, workflowInfo.name as WName
	FROM w_node
	JOIN workflowInfo ON workflowInfo.id = w_node.workflow_id
	WHERE w_node.id < workflowInfo.rootNodeID
)
DELETE FROM w_node_trigger where child_node_id IN (SELECT nodeID FROM oldNode);


-- clean w_node
WITH workflowInfo AS (
	SELECT id, name, CAST(workflow_data->'node'->>'id' AS BIGINT) as rootNodeID
	FROM workflow
),
oldNode as (
	SELECT w_node.id as nodeID, w_node.name as nodeName, workflowInfo.id as wID, workflowInfo.name as WName
	FROM w_node
	JOIN workflowInfo ON workflowInfo.id = w_node.workflow_id
	WHERE w_node.id < workflowInfo.rootNodeID
)
DELETE FROM w_node where id IN (SELECT nodeID FROM oldNode);

-- +migrate Down
SELECT 1;
