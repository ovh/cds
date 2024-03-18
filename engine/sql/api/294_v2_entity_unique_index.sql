-- +migrate Up
DROP INDEX idx_unq_repo_branch_type_name;
SELECT create_unique_index('entity', 'idx_unq_repo_branch_type_name_commit', 'project_repository_id,ref,type,name,commit');
-- +migrate Down
DROP INDEX IF EXISTS idx_unq_repo_branch_type_name;
SELECT create_unique_index('entity', 'idx_unq_repo_branch_type_name', 'project_repository_id,ref,type,name');
