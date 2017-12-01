-- +migrate Up
ALTER TABLE workflow_hook_model DROP COLUMN icon;
ALTER TABLE workflow_hook_model ADD COLUMN icon VARCHAR(50);
ALTER TABLE workflow_hook_model DROP COLUMN image;

UPDATE workflow_hook_model SET icon = 'fa-anchor' WHERE name= 'WebHook';
UPDATE workflow_hook_model SET icon = 'git square' WHERE name= 'Git Repository Poller';
UPDATE workflow_hook_model SET icon = 'fa-clock-o' WHERE name= 'Scheduler';

-- +migrate Down
ALTER TABLE workflow_hook_model DROP COLUMN icon;
ALTER TABLE workflow_hook_model ADD COLUMN icon bytea;
ALTER TABLE workflow_hook_model ADD COLUMN image string;