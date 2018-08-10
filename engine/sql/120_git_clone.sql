-- +migrate Up

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'GitClone' and type = 'Builtin'), 'depth', 'gitClone use a depth of 50 by default. You can remove --depth with the value ''false''', 'string', '', true);
INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'GitClone' and type = 'Builtin'), 'submodules', 'gitClone clones submodules by default, you can set ''false'' to avoid this', 'boolean', 'true', true);

UPDATE action_parameter set value = '' where (name='url' or name = 'privateKey') and action_id=(select id from action where name = 'GitClone' and type = 'Builtin');

-- +migrate Down

DELETE from action_parameter where name = 'depth' and action_id = (select id from action where name = 'GitClone' and type = 'Builtin');
DELETE from action_parameter where name = 'submodules' and action_id = (select id from action where name = 'GitClone' and type = 'Builtin');

UPDATE action_parameter set value = '{{.git.url}}' where name='url' and action_id=(select id from action where name = 'GitClone' and type = 'Builtin');
