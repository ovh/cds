-- +migrate Up
ALTER TABLE integration_model RENAME COLUMN file_storage TO storage;
ALTER TABLE integration_model DROP COLUMN block_storage;

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'Artifact Upload' and type = 'Builtin'), 'destination', 'Destination of this artifact. Use the name of integration attached on your project', 'string', '', true);

-- +migrate Down

DELETE from action_parameter where name = 'destination' and action_id = (select id from action where name = 'Artifact Upload' and type = 'Builtin');

ALTER TABLE integration_model RENAME COLUMN storage TO file_storage;
ALTER TABLE integration_model ADD COLUMN block_storage BOOLEAN default false;
