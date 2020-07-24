-- +migrate Up
UPDATE action_edge SET enabled = false WHERE action_edge.id IN (SELECT action_edge.id FROM action_edge_parameter
    JOIN action_edge ON action_edge_parameter.action_edge_id = action_edge.id
    JOIN action ON action_edge.child_id = action.id
    WHERE action_edge_parameter.name = 'enabled' AND action_edge_parameter.value = 'false' AND (action.name = 'Artifact Download' OR action.name = 'Artifact Upload'));

-- +migrate Down
SELECT 1;