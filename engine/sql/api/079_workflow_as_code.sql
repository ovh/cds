-- +migrate Up
ALTER TABLE workflow ADD COLUMN from_repository TEXT DEFAULT '';
ALTER TABLE workflow ADD COLUMN derived_from_workflow_id BIGINT DEFAULT 0;
ALTER TABLE workflow ADD COLUMN derived_from_workflow_name TEXT DEFAULT '';
ALTER TABLE workflow ADD COLUMN derivation_branch TEXT DEFAULT '';

UPDATE workflow SET from_repository = '';
UPDATE workflow SET derived_from_workflow_id = 0;
UPDATE workflow SET derived_from_workflow_name = '';
UPDATE workflow SET derivation_branch = '';

-- +migrate Down
ALTER TABLE workflow DROP COLUMN from_repository;
ALTER TABLE workflow DROP COLUMN derived_from_workflow_id;
ALTER TABLE workflow DROP COLUMN derived_from_workflow_name;
ALTER TABLE workflow DROP COLUMN derivation_branch;