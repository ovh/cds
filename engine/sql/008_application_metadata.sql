-- +migrate Up
ALTER TABLE application ADD COLUMN metadata JSONB;

ALTER TABLE application ALTER COLUMN project_id SET NOT NULL ;

ALTER TABLE application ALTER COLUMN name SET NOT NULL ;
ALTER TABLE application ALTER COLUMN name SET DEFAULT '';

UPDATE application set description = '' where description IS NULL;
ALTER TABLE application ALTER COLUMN description SET DEFAULT '';
ALTER TABLE application ALTER COLUMN description SET NOT NULL;

UPDATE application set repo_fullname = '' where repo_fullname IS NULL;
ALTER TABLE application ALTER COLUMN repo_fullname SET DEFAULT '';
ALTER TABLE application ALTER COLUMN repo_fullname SET NOT NULL;

-- +migrate Down
ALTER table application DROP COLUMN metadata;