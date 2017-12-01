-- +migrate Up
INSERT INTO action_parameter(action_id, name, type, value, description) VALUES ((select id from action where name = 'Artifact Download'), 'pattern', 'string', '', 'Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB)');

-- +migrate Down
DELETE FROM action_parameter where name='pattern' and action_id = (select id from action where name = 'Artifact Download');
