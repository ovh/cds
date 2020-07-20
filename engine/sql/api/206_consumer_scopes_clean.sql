-- +migrate Up

-- Clean for 153_access_token.sql
DROP TABLE IF EXISTS "access_token_group";
DROP TABLE IF EXISTS "access_token";

-- Clean for 156_integration.sql
ALTER TABLE "application_deployment_strategy" DROP COLUMN IF EXISTS project_platform_id;

-- Clean for 158_workflow_node_permission.sql
DROP TABLE IF EXISTS "workflow_group";
DROP TABLE IF EXISTS "application_group";
DROP TABLE IF EXISTS "environment_group";
DROP TABLE IF EXISTS "pipeline_group";

-- Clean for 180_user.sql
DROP TABLE IF EXISTS "authentified_user_migration";
DROP TABLE IF EXISTS "old_worker";
DROP TABLE IF EXISTS "old_services";

-- Clean for 183_drop_deprecated_user.sql
DROP TABLE IF EXISTS "user_persistent_session";
DROP TABLE IF EXISTS "group_user";
DROP TABLE IF EXISTS "pipeline_build";
ALTER TABLE "project_favorite" DROP COLUMN IF EXISTS user_id;
ALTER TABLE "workflow_favorite" DROP COLUMN IF EXISTS user_id;
ALTER TABLE "broadcast_read" DROP COLUMN IF EXISTS user_id;
ALTER TABLE "user_timeline" DROP COLUMN IF EXISTS user_id;
ALTER TABLE "workflow_template_bulk" DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS "user";

-- Clean for 185_consumer_scopes.sql
ALTER TABLE "auth_consumer" DROP COLUMN scopes;

-- Clean for 186_application_key.sql
DROP TABLE IF EXISTS "application_key_tmp";

-- Clean for 187_project_key.sql
DROP TABLE IF EXISTS "project_key_tmp";

-- Clean for 189_environment_key.sql
DROP TABLE IF EXISTS "environment_key_tmp";

-- Clean for 190_application_variables.sql
DROP TABLE IF EXISTS "application_variable_tmp";

-- Clean for 191_project_variables.sql
DROP TABLE IF EXISTS "project_variable_tmp";

-- Clean for 192_environment_variables.sql
DROP TABLE IF EXISTS "environment_variable_tmp";

-- Clean for 198_app_crypto.sql
ALTER TABLE "application" DROP COLUMN IF EXISTS repositories_manager_id;
ALTER TABLE "application" DROP COLUMN IF EXISTS vcs_strategy;
ALTER TABLE "application_deployment_strategy" DROP COLUMN IF EXISTS config;

-- Clean for 059_repositoriesmanager_project.sql
DROP TABLE IF EXISTS "repositories_manager_project";
DROP TABLE IF EXISTS "repositories_manager";

-- Clean for 199_refact_integration_crypto.sql
ALTER TABLE "project_integration" DROP COLUMN IF EXISTS config;
ALTER TABLE "integration_model" DROP COLUMN IF EXISTS public_configurations;

-- Clean for 204_project_vcs_server.sql
ALTER TABLE "project" DROP COLUMN IF EXISTS vcs_servers;

-- +migrate Down

-- Clean for 059_repositoriesmanager_project.sql
-- Used in delete project before this migration.
CREATE TABLE IF NOT EXISTS "repositories_manager_project" ( id_repositories_manager BIGINT NOT NULL, id_project BIGINT NOT NULL, data JSONB, PRIMARY KEY(id_repositories_manager, id_project));

-- Clean for 185_consumer_scopes.sql
ALTER TABLE "auth_consumer" ADD COLUMN IF NOT EXISTS scopes JSONB;
UPDATE auth_consumer SET scopes = scope_details WHERE scope_details::TEXT = 'null';
UPDATE auth_consumer SET scopes = tmp1.scopes FROM (
	SELECT tmp2.id AS id, json_agg(tmp2."scope_detail"->>'scope') AS scopes
  FROM (
		SELECT id, jsonb_array_elements(scope_details::JSONB) AS "scope_detail" FROM auth_consumer WHERE auth_consumer.scope_details::TEXT <> 'null'
	) AS tmp2 GROUP BY tmp2.id
) AS tmp1
WHERE auth_consumer.id = tmp1.id AND auth_consumer.scope_details::TEXT <> 'null';

-- Clean for 204_project_vcs_server.sql
-- Used for migration before the migration.
ALTER TABLE "project" ADD COLUMN IF NOT EXISTS vcs_servers JSONB;
