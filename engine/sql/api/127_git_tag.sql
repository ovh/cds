-- +migrate Up

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'GitTag' and type = 'Builtin'), 'prefix', 'Prefix for tag name', 'string', '', true);

-- +migrate Down

DELETE from action_parameter where name = 'prefix' and action_id = (select id from action where name = 'GitTag' and type = 'Builtin');

