-- The run search of a project reads the runs of one project in the order it sorts them on, and takes
-- a page of them. Without these the planner reads and sorts every run of the project on each search.
-- One index per sort column serves both directions: it is scanned backwards for the descending
-- order, which puts the nulls on the side the ORDER BY expects either way.

-- +migrate Up
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_project_started', 'project_key,started');
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_project_last_modified', 'project_key,last_modified');

-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_project_started;
DROP INDEX IF EXISTS idx_v2_workflow_run_project_last_modified;
