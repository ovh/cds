-- +migrate Up

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'GitTag' and type = 'Builtin'), 'vprefix', 'add ''v'' prefix in tag created', 'boolean', 'false', true);

-- +migrate Down

DELETE from action_parameter where name = 'vprefix' and action_id = (select id from action where name = 'GitTag' and type = 'Builtin');

