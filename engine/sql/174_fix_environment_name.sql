-- +migrate Up
UPDATE environment SET name = REPLACE(name,'/','_') WHERE name like '%/%';

-- +migrate Down
SELECT 1;
