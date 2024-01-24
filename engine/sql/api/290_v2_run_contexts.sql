-- +migrate Up
CREATE TABLE old_v2_workflow_run_context AS select id, contexts from v2_workflow_run;
ALTER TABLE old_v2_workflow_run_context ADD PRIMARY KEY (id);
ALTER TABLE v2_workflow_run DROP COLUMN contexts;
ALTER TABLE v2_workflow_run ADD COLUMN contexts JSONB;

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN contexts;
ALTER TABLE v2_workflow_run ADD COLUMN contexts TEXT;

UPDATE v2_workflow_run SET contexts = old_v2_workflow_run_context.contexts
FROM old_v2_workflow_run_context WHERE v2_workflow_run.id = old_v2_workflow_run_context.id;

DROP TABLE old_v2_workflow_run_context;
