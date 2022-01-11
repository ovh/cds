-- +migrate Up
UPDATE action SET name = 'ReleaseVCS' WHERE name = 'Release' and type = 'Builtin';

-- +migrate Down
UPDATE action SET name = 'Release' WHERE name = 'ReleaseVCS' and type = 'Builtin';
