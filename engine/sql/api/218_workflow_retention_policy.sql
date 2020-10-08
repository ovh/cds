-- +migrate Up
alter table "workflow" add column retention_policy TEXT;
update workflow set retention_policy = 'return (git_branch_exist == "false" and run_days_before < 2) or run_days_before < 365';
alter table "workflow" alter column retention_policy SET NOT NULL;

alter table "workflow" add column max_runs INT;
update workflow set max_runs = 255;
alter table "workflow" alter column max_runs SET NOT NULL;
alter table "workflow" alter column max_runs SET DEFAULT 255;

-- +migrate Down
alter table "workflow" drop column retention_policy;
