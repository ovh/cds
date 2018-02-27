-- +migrate Up
DELETE FROM environment_group where environment_id=1;

-- +migrate Down
SELECT 1;
