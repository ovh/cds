-- +migrate Up
UPDATE action_parameter
  SET type = 'ssh-key', value = ''
WHERE action_id = (
  SELECT id
    FROM action
    WHERE name = 'GitClone' AND type = 'Builtin'
) AND name = 'privateKey';

-- +migrate Down
UPDATE action_parameter
  SET type = 'string', value = '{{.cds.app.key}}'
WHERE action_id = (
  SELECT id
    FROM action
    WHERE name = 'GitClone' AND type = 'Builtin'
) AND name = 'privateKey';
