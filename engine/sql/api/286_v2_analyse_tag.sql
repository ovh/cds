-- +migrate Up
ALTER TABLE project_repository_analysis RENAME COLUMN "branch" to "ref";
UPDATE project_repository_analysis SET ref = 'refs/heads/' || ref;

ALTER TABLE entity RENAME COLUMN "branch" to "ref";
UPDATE entity SET ref = 'refs/heads/' || ref;

UPDATE v2_workflow_run SET workflow_ref = 'refs/heads/' || workflow_ref;

ALTER TABLE v2_workflow_hook RENAME COLUMN "branch" to "ref";
UPDATE v2_workflow_hook SET ref = 'refs/heads/' || ref;


-- +migrate Down
ALTER TABLE entity RENAME COLUMN "ref" to "branch";
ALTER TABLE project_repository_analysis RENAME COLUMN "ref" to "branch";
ALTER TABLE v2_workflow_hook RENAME COLUMN "ref" to "branch";

UPDATE project_repository_analysis SET branch = ltrim(branch, 'refs/heads/');
UPDATE entity SET branch = ltrim(branch, 'refs/heads/');
UPDATE v2_workflow_run SET workflow_ref = ltrim(workflow_ref, 'refs/heads/');
UPDATE v2_workflow_hook SET branch = ltrim(branch, 'refs/heads/');


