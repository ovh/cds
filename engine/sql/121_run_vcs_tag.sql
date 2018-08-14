-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN vcs_tag TEXT;
INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'GitClone' and type = 'Builtin'), 'tag', 'Useful when you want to git clone a specific tag', 'string', '', true);

-- +migrate Down
DELETE from action_parameter where name = 'tag' and action_id = (select id from action where name = 'GitClone' and type = 'Builtin');
ALTER TABLE workflow_node_run DROP COLUMN vcs_tag;
