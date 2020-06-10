-- +migrate Up

DELETE FROM action_requirement WHERE type = 'network';

-- +migrate Down

SELECT 1;

