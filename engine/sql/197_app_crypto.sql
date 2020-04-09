-- +migrate Up

-- RefactorAppDeploymentStrategies
CREATE TABLE application_deployment_strategy_tmp AS SELECT * FROM application_deployment_strategy;
ALTER TABLE application_deployment_strategy_tmp ADD PRIMARY KEY (application_id, project_platform_id);
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS id SERIAL;
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS cipher_config BYTEA;
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application_deployment_strategy" ADD COLUMN IF NOT EXISTS signer TEXT;

-- RefactorApplicationVCS
CREATE TABLE application_tmp AS SELECT * FROM "application";
ALTER TABLE application_tmp ADD PRIMARY KEY (id);
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS cipher_vcs_strategy BYTEA;
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application" ADD COLUMN IF NOT EXISTS signer TEXT;


-- +migrate Down

ALTER TABLE "application_deployment_strategy" DROP COLUMN migrate;
UPDATE  application_deployment_strategy 
SET     config = application_deployment_strategy_tmp.config
FROM    application_deployment_strategy_tmp
WHERE   application_deployment_strategy_tmp.application_id = application_deployment_strategy.application_id
AND     application_deployment_strategy_tmp.project_platform_id = application_deployment_strategy.project_platform_id;
DROP TABLE application_deployment_strategy_tmp;

ALTER TABLE "application" DROP COLUMN migrate;
UPDATE  "application" 
SET     vcs_strategy = application_tmp.vcs_strategy
FROM    application_tmp
WHERE   application_tmp.id = "application".id
DROP TABLE application_tmp;
