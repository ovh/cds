-- +migrate Up

-- RefactorAppDeploymentStrategies
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS id SERIAL;
ALTER TABLE "application_deployment_strategy" ALTER COLUMN id SET NOT NULL;
ALTER TABLE "application_deployment_strategy" DROP CONSTRAINT application_deployment_strategy_pkey;
ALTER TABLE "application_deployment_strategy" ADD PRIMARY KEY (id);
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS cipher_config BYTEA;
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS signer TEXT;

-- RefactorApplicationVCS
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS cipher_vcs_strategy BYTEA;
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS signer TEXT;


-- +migrate Down

ALTER TABLE "application_deployment_strategy" DROP COLUMN id;
ALTER TABLE "application_deployment_strategy" DROP COLUMN cipher_config;
ALTER TABLE "application_deployment_strategy" DROP COLUMN sig;
ALTER TABLE "application_deployment_strategy" DROP COLUMN signer;
ALTER TABLE "application_deployment_strategy" ADD PRIMARY KEY (application_id, project_integration_id);


ALTER TABLE "application" DROP COLUMN cipher_vcs_strategy;
ALTER TABLE "application" DROP COLUMN sig;
ALTER TABLE "application" DROP COLUMN signer;
