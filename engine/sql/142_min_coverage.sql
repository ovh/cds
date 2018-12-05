-- +migrate Up

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'Coverage' and type = 'Builtin'), 'minimum', 'Minimum percentage of coverage required (-1 means no minimum).', 'string', '', true);

-- +migrate Down

DELETE from action_parameter where name = 'minimum' and action_id = (select id from action where name = 'Coverage' and type = 'Builtin');
