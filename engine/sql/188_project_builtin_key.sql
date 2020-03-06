-- +migrate Up
UPDATE project_key SET builtin = true WHERE type = 'pgp' AND name = 'builtin';

-- +migrate Down
SELECT 1;

