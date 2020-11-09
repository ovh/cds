-- +migrate Up
DELETE FROM action_requirement where type = 'volume';

-- +migrate Down
SELECT 1;
