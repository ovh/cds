-- +migrate Up
UPDATE workflow SET retention_policy = '' WHERE retention_policy='return (git_branch_exist == "false" and run_days_before < 2) or run_days_before < 365';

-- +migrate Down
SELECT 1;
