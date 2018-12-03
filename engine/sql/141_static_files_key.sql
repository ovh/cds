-- +migrate Up

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'Serve Static Files' and type = 'Builtin'), 'static-key', 'Indicate a static-key which will be a reference to keep the same generated URL. Example: {{.git.branch}}', 'string', '', true);
ALTER TABLE workflow_node_run_static_files ADD COLUMN static_key TEXT DEFAULT '';

-- +migrate Down

DELETE from action_parameter where name = 'static-key' and action_id = (select id from action where name = 'Serve Static Files' and type = 'Builtin');
ALTER TABLE workflow_node_run_static_files DROP COLUMN static_key;
