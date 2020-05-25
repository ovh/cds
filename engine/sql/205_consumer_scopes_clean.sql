-- +migrate Up

-- Clean for 010_audit.sql
DROP TABLE IF EXISTS "application_variable_audit_old";

-- Clean for 153_access_token.sql
DROP TABLE IF EXISTS "access_token_group";
DROP TABLE IF EXISTS "access_token";

-- Clean for 156_integration.sql
ALTER TABLE "application_deployment_strategy" DROP COLUMN IF EXISTS project_platform_id;

-- Clean for 180_user.sql
DROP TABLE IF EXISTS "authentified_user_migration";

-- Clean for 183_drop_deprecated_user.sql
DROP TABLE IF EXISTS "user";
DROP TABLE IF EXISTS "user_persistent_session";

-- Clean for 185_consumer_scopes.sql
ALTER TABLE "auth_consumer" DROP COLUMN scopes;

-- Clean for 186_application_key.sql
DROP TABLE IF EXISTS "application_key_tmp";

-- Clean for 190_application_variables.sql
DROP TABLE IF EXISTS "application_variable_tmp";

-- Clean for 198_app_crypto.sql
ALTER TABLE "application" DROP COLUMN IF EXISTS repositories_manager_id;
ALTER TABLE "application" DROP COLUMN IF EXISTS vcs_strategy;
ALTER TABLE "application_deployment_strategy" DROP COLUMN IF EXISTS config;


-- +migrate Down

-- Clean for 185_consumer_scopes.sql
ALTER TABLE "auth_consumer" ADD COLUMN scopes JSONB;
UPDATE auth_consumer SET scopes = scope_details WHERE scope_details::TEXT = 'null';
UPDATE auth_consumer SET scopes = tmp1.scopes FROM (
	SELECT tmp2.id AS id, json_agg(tmp2."scope_detail"->>'scope') AS scopes
  FROM (
		SELECT id, jsonb_array_elements(scope_details::JSONB) AS "scope_detail" FROM auth_consumer WHERE auth_consumer.scope_details::TEXT <> 'null'
	) AS tmp2 GROUP BY tmp2.id
) AS tmp1
WHERE auth_consumer.id = tmp1.id AND auth_consumer.scope_details::TEXT <> 'null';
