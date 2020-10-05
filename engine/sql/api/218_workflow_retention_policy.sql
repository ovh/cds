-- +migrate Up
alter table "workflow" add column retention_policy TEXT;
update workflow set retention_policy = 'return (git_branch_exist == "false" and run_date_before < 2) or run_date_before < 365';
alter table "workflow" alter column retention_policy SET NOT NULL;

-- +migrate Down
alter table "workflow" drop column retention_policy;
