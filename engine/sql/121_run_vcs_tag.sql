-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN vcs_tag TEXT;
INSERT into action_parameter (action_id, name, description, type, value, advanced) (SELECT id, 'tag' AS name, 'Useful when you want to git clone a specific tag' AS description, 'string' AS type, '' AS value, true AS advanced from action where name = 'GitClone' and type = 'Builtin');

-- +migrate Down
DELETE from action_parameter where name = 'tag' and action_id = (select id from action where name = 'GitClone' and type = 'Builtin');
ALTER TABLE workflow_node_run DROP COLUMN vcs_tag;
