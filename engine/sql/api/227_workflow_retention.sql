-- +migrate Up
UPDATE workflow SET retention_policy = '' WHERE retention_policy='return (git_branch_exist == "false" and run_days_before < 2) or run_days_before < 365';

-- +migrate Down
UPDATE workflow SET retention_policy = 'return (git_branch_exist == "false" and run_days_before < 2) or (git_branch_exist == "true" and run_days_before < 365)' WHERE retention_policy = ''
