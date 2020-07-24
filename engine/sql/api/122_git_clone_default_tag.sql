-- +migrate Up
UPDATE action_parameter SET value = '{{.git.tag}}' WHERE name='tag' AND action_id=(select id from action where name = 'GitClone' and type = 'Builtin');

-- +migrate Down
UPDATE action_parameter SET value = '' WHERE name='tag' AND action_id=(select id from action where name = 'GitClone' and type = 'Builtin');
