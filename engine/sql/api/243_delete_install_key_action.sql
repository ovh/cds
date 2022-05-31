-- +migrate Up
DELETE FROM "action" WHERE name = 'InstallKey' and type = 'Builtin';

-- +migrate Down
SELECT 1;
