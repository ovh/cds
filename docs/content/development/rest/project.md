+++
title = "project"
+++


### POST `/project`

URL         | **`/project`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addProjectHandler%22)
    









### GET `/project`

URL         | **`/project`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getProjectsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProjectsHandler%22)
    









### DELETE `/project/<project-key>`

URL         | **`/project/<project-key>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteProjectHandler%22)
    









### GET `/project/<project-key>`

URL         | **`/project/<project-key>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProjectHandler%22)
    









### PUT `/project/<project-key>`

URL         | **`/project/<project-key>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateProjectHandler%22)
    









### GET `/project/<project-key>/all/keys`

URL         | **`/project/<project-key>/all/keys`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getAllKeysProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAllKeysProjectHandler%22)
    









### DELETE `/project/<project-key>/application/<applicationName>`

URL         | **`/project/<project-key>/application/<applicationName>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>`

URL         | **`/project/<project-key>/application/<applicationName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationHandler%22)
    









### PUT `/project/<project-key>/application/<applicationName>`

URL         | **`/project/<project-key>/application/<applicationName>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateApplicationHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/clone`

URL         | **`/project/<project-key>/application/<applicationName>/clone`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [cloneApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+cloneApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/deployment/config`

URL         | **`/project/<project-key>/application/<applicationName>/deployment/config`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationDeploymentStrategiesConfigHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationDeploymentStrategiesConfigHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/deployment/config/<integration>`

URL         | **`/project/<project-key>/application/<applicationName>/deployment/config/<integration>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postApplicationDeploymentStrategyConfigHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postApplicationDeploymentStrategyConfigHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/deployment/config/<integration>`

URL         | **`/project/<project-key>/application/<applicationName>/deployment/config/<integration>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationDeploymentStrategyConfigHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationDeploymentStrategyConfigHandler%22)
    









### DELETE `/project/<project-key>/application/<applicationName>/deployment/config/<integration>`

URL         | **`/project/<project-key>/application/<applicationName>/deployment/config/<integration>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteApplicationDeploymentStrategyConfigHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteApplicationDeploymentStrategyConfigHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/keys`

URL         | **`/project/<project-key>/application/<applicationName>/keys`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getKeysInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getKeysInApplicationHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/keys`

URL         | **`/project/<project-key>/application/<applicationName>/keys`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addKeyInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addKeyInApplicationHandler%22)
    









### DELETE `/project/<project-key>/application/<applicationName>/keys/<name>`

URL         | **`/project/<project-key>/application/<applicationName>/keys/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteKeyInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteKeyInApplicationHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/metadata/<metadata>`

URL         | **`/project/<project-key>/application/<applicationName>/metadata/<metadata>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postApplicationMetadataHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postApplicationMetadataHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/metrics/<metricName>`

URL         | **`/project/<project-key>/application/<applicationName>/metrics/<metricName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationMetricHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationMetricHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/variable`

URL         | **`/project/<project-key>/application/<applicationName>/variable`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariablesInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariablesInApplicationHandler%22)
    









### PUT `/project/<project-key>/application/<applicationName>/variable/<name>`

URL         | **`/project/<project-key>/application/<applicationName>/variable/<name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateVariableInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateVariableInApplicationHandler%22)
    









### DELETE `/project/<project-key>/application/<applicationName>/variable/<name>`

URL         | **`/project/<project-key>/application/<applicationName>/variable/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteVariableFromApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteVariableFromApplicationHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/variable/<name>`

URL         | **`/project/<project-key>/application/<applicationName>/variable/<name>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addVariableInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addVariableInApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/variable/<name>`

URL         | **`/project/<project-key>/application/<applicationName>/variable/<name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableInApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/variable/<name>/audit`

URL         | **`/project/<project-key>/application/<applicationName>/variable/<name>/audit`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableAuditInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableAuditInApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/variable/audit`

URL         | **`/project/<project-key>/application/<applicationName>/variable/audit`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariablesAuditInApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariablesAuditInApplicationHandler%22)
    









### GET `/project/<project-key>/application/<applicationName>/vcsinfos`

URL         | **`/project/<project-key>/application/<applicationName>/vcsinfos`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationVCSInfosHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationVCSInfosHandler%22)
    









### POST `/project/<project-key>/application/<applicationName>/vulnerability/<id>`

URL         | **`/project/<project-key>/application/<applicationName>/vulnerability/<id>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postVulnerabilityHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postVulnerabilityHandler%22)
    









### GET `/project/<project-key>/applications`

URL         | **`/project/<project-key>/applications`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationsHandler%22)
    









### POST `/project/<project-key>/applications`

URL         | **`/project/<project-key>/applications`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addApplicationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addApplicationHandler%22)
    









### POST `/project/<project-key>/encrypt`

URL         | **`/project/<project-key>/encrypt`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postEncryptVariableHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postEncryptVariableHandler%22)
    









### POST `/project/<project-key>/environment`

URL         | **`/project/<project-key>/environment`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addEnvironmentHandler%22)
    









### GET `/project/<project-key>/environment`

URL         | **`/project/<project-key>/environment`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getEnvironmentsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getEnvironmentsHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>`

URL         | **`/project/<project-key>/environment/<environmentName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getEnvironmentHandler%22)
    









### DELETE `/project/<project-key>/environment/<environmentName>`

URL         | **`/project/<project-key>/environment/<environmentName>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteEnvironmentHandler%22)
    









### PUT `/project/<project-key>/environment/<environmentName>`

URL         | **`/project/<project-key>/environment/<environmentName>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateEnvironmentHandler%22)
    









### POST `/project/<project-key>/environment/<environmentName>/clone/<cloneName>`

URL         | **`/project/<project-key>/environment/<environmentName>/clone/<cloneName>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [cloneEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+cloneEnvironmentHandler%22)
    









### POST `/project/<project-key>/environment/<environmentName>/keys`

URL         | **`/project/<project-key>/environment/<environmentName>/keys`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addKeyInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addKeyInEnvironmentHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>/keys`

URL         | **`/project/<project-key>/environment/<environmentName>/keys`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getKeysInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getKeysInEnvironmentHandler%22)
    









### DELETE `/project/<project-key>/environment/<environmentName>/keys/<name>`

URL         | **`/project/<project-key>/environment/<environmentName>/keys/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteKeyInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteKeyInEnvironmentHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>/usage`

URL         | **`/project/<project-key>/environment/<environmentName>/usage`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getEnvironmentUsageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getEnvironmentUsageHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>/variable`

URL         | **`/project/<project-key>/environment/<environmentName>/variable`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariablesInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariablesInEnvironmentHandler%22)
    









### DELETE `/project/<project-key>/environment/<environmentName>/variable/<name>`

URL         | **`/project/<project-key>/environment/<environmentName>/variable/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteVariableFromEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteVariableFromEnvironmentHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>/variable/<name>`

URL         | **`/project/<project-key>/environment/<environmentName>/variable/<name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableInEnvironmentHandler%22)
    









### POST `/project/<project-key>/environment/<environmentName>/variable/<name>`

URL         | **`/project/<project-key>/environment/<environmentName>/variable/<name>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addVariableInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addVariableInEnvironmentHandler%22)
    









### PUT `/project/<project-key>/environment/<environmentName>/variable/<name>`

URL         | **`/project/<project-key>/environment/<environmentName>/variable/<name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateVariableInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateVariableInEnvironmentHandler%22)
    









### GET `/project/<project-key>/environment/<environmentName>/variable/<name>/audit`

URL         | **`/project/<project-key>/environment/<environmentName>/variable/<name>/audit`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableAuditInEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableAuditInEnvironmentHandler%22)
    









### POST `/project/<project-key>/environment/import`

URL         | **`/project/<project-key>/environment/import`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [importNewEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importNewEnvironmentHandler%22)
    









### POST `/project/<project-key>/environment/import/<environmentName>`

URL         | **`/project/<project-key>/environment/import/<environmentName>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [importIntoEnvironmentHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importIntoEnvironmentHandler%22)
    









### GET `/project/<project-key>/export/application/<applicationName>`

URL         | **`/project/<project-key>/export/application/<applicationName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getApplicationExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getApplicationExportHandler%22)
    









### GET `/project/<project-key>/export/environment/<environmentName>`

URL         | **`/project/<project-key>/export/environment/<environmentName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getEnvironmentExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getEnvironmentExportHandler%22)
    









### GET `/project/<project-key>/export/pipeline/<pipelineKey>`

URL         | **`/project/<project-key>/export/pipeline/<pipelineKey>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getPipelineExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPipelineExportHandler%22)
    









### GET `/project/<project-key>/export/workflows/<workflow-name>`

URL         | **`/project/<project-key>/export/workflows/<workflow-name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowExportHandler%22)
    









### POST `/project/<project-key>/group`

URL         | **`/project/<project-key>/group`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addGroupInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addGroupInProjectHandler%22)
    









### DELETE `/project/<project-key>/group/<group>`

URL         | **`/project/<project-key>/group/<group>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteGroupFromProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteGroupFromProjectHandler%22)
    









### PUT `/project/<project-key>/group/<group>`

URL         | **`/project/<project-key>/group/<group>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateGroupRoleOnProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateGroupRoleOnProjectHandler%22)
    









### POST `/project/<project-key>/group/import`

URL         | **`/project/<project-key>/group/import`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [importGroupsInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importGroupsInProjectHandler%22)
    









### POST `/project/<project-key>/import/application`

URL         | **`/project/<project-key>/import/application`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postApplicationImportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postApplicationImportHandler%22)
    









### import an environment yml file

URL         | **`/project/<project-key>/import/environment`**
----------- |----------
Method      | POST     
Query Parameter | force=true or false. If false and if the environment already exists, raise an error
Permissions |  Auth: true
Code        | [postEnvironmentImportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postEnvironmentImportHandler%22)
    

#### Description
import an environment yml file with `cdsctl environment import myenv.env.yml`







### POST `/project/<project-key>/import/pipeline`

URL         | **`/project/<project-key>/import/pipeline`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [importPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importPipelineHandler%22)
    









### PUT `/project/<project-key>/import/pipeline/<pipelineKey>`

URL         | **`/project/<project-key>/import/pipeline/<pipelineKey>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putImportPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putImportPipelineHandler%22)
    









### POST `/project/<project-key>/import/workflows`

URL         | **`/project/<project-key>/import/workflows`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowImportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowImportHandler%22)
    









### PUT `/project/<project-key>/import/workflows/<workflow-name>`

URL         | **`/project/<project-key>/import/workflows/<workflow-name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putWorkflowImportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putWorkflowImportHandler%22)
    









### POST `/project/<project-key>/integrations`

URL         | **`/project/<project-key>/integrations`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postProjectIntegrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postProjectIntegrationHandler%22)
    









### GET `/project/<project-key>/integrations`

URL         | **`/project/<project-key>/integrations`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getProjectIntegrationsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProjectIntegrationsHandler%22)
    









### DELETE `/project/<project-key>/integrations/<integrationName>`

URL         | **`/project/<project-key>/integrations/<integrationName>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteProjectIntegrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteProjectIntegrationHandler%22)
    









### PUT `/project/<project-key>/integrations/<integrationName>`

URL         | **`/project/<project-key>/integrations/<integrationName>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putProjectIntegrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putProjectIntegrationHandler%22)
    









### GET `/project/<project-key>/integrations/<integrationName>`

URL         | **`/project/<project-key>/integrations/<integrationName>`**
----------- |----------
Method      | GET     
Permissions |  AllowServices: true -  Auth: true
Code        | [getProjectIntegrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProjectIntegrationHandler%22)
    









### POST `/project/<project-key>/keys`

URL         | **`/project/<project-key>/keys`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addKeyInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addKeyInProjectHandler%22)
    









### GET `/project/<project-key>/keys`

URL         | **`/project/<project-key>/keys`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getKeysInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getKeysInProjectHandler%22)
    









### DELETE `/project/<project-key>/keys/<name>`

URL         | **`/project/<project-key>/keys/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteKeyInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteKeyInProjectHandler%22)
    









### PUT `/project/<project-key>/labels`

URL         | **`/project/<project-key>/labels`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putProjectLabelsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putProjectLabelsHandler%22)
    









### DEPRECATED

URL         | **`/project/<project-key>/notifications`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getProjectNotificationsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProjectNotificationsHandler%22)
    









### GET `/project/<project-key>/pipeline`

URL         | **`/project/<project-key>/pipeline`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getPipelinesHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPipelinesHandler%22)
    









### POST `/project/<project-key>/pipeline`

URL         | **`/project/<project-key>/pipeline`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addPipelineHandler%22)
    









### GET `/project/<project-key>/pipeline/<pipelineKey>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPipelineHandler%22)
    









### DELETE `/project/<project-key>/pipeline/<pipelineKey>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deletePipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deletePipelineHandler%22)
    









### PUT `/project/<project-key>/pipeline/<pipelineKey>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updatePipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updatePipelineHandler%22)
    









### GET `/project/<project-key>/pipeline/<pipelineKey>/audits`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/audits`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getPipelineAuditHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPipelineAuditHandler%22)
    









### GET `/project/<project-key>/pipeline/<pipelineKey>/parameter`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/parameter`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getParametersInPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getParametersInPipelineHandler%22)
    









### POST `/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addParameterInPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addParameterInPipelineHandler%22)
    









### DELETE `/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteParameterFromPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteParameterFromPipelineHandler%22)
    









### PUT `/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/parameter/<name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateParameterInPipelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateParameterInPipelineHandler%22)
    









### POST `/project/<project-key>/pipeline/<pipelineKey>/rollback/<auditID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/rollback/<auditID>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postPipelineRollbackHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPipelineRollbackHandler%22)
    









### POST `/project/<project-key>/pipeline/<pipelineKey>/stage`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addStageHandler%22)
    









### DELETE `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteStageHandler%22)
    









### PUT `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateStageHandler%22)
    









### GET `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getStageHandler%22)
    









### POST `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addJobToStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addJobToStageHandler%22)
    









### DELETE `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job/<jobID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job/<jobID>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteJobHandler%22)
    









### PUT `/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job/<jobID>`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/<stageID>/job/<jobID>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateJobHandler%22)
    









### POST `/project/<project-key>/pipeline/<pipelineKey>/stage/move`

URL         | **`/project/<project-key>/pipeline/<pipelineKey>/stage/move`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [moveStageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+moveStageHandler%22)
    









### POST `/project/<project-key>/preview/pipeline`

URL         | **`/project/<project-key>/preview/pipeline`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postPipelinePreviewHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPipelinePreviewHandler%22)
    









### POST `/project/<project-key>/preview/workflows`

URL         | **`/project/<project-key>/preview/workflows`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowPreviewHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowPreviewHandler%22)
    









### Pull is only in yaml

URL         | **`/project/<project-key>/pull/workflows/<workflow-name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowPullHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowPullHandler%22)
    









### POST `/project/<project-key>/push/workflows`

URL         | **`/project/<project-key>/push/workflows`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowPushHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowPushHandler%22)
    









### GET `/project/<project-key>/repositories_manager`

URL         | **`/project/<project-key>/repositories_manager`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getRepositoriesManagerForProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getRepositoriesManagerForProjectHandler%22)
    









### DELETE `/project/<project-key>/repositories_manager/<name>`

URL         | **`/project/<project-key>/repositories_manager/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteRepositoriesManagerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteRepositoriesManagerHandler%22)
    









### POST `/project/<project-key>/repositories_manager/<name>/application/<applicationName>/attach`

URL         | **`/project/<project-key>/repositories_manager/<name>/application/<applicationName>/attach`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [attachRepositoriesManagerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+attachRepositoriesManagerHandler%22)
    









### POST `/project/<project-key>/repositories_manager/<name>/application/<applicationName>/detach`

URL         | **`/project/<project-key>/repositories_manager/<name>/application/<applicationName>/detach`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [detachRepositoriesManagerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+detachRepositoriesManagerHandler%22)
    









### POST `/project/<project-key>/repositories_manager/<name>/authorize`

URL         | **`/project/<project-key>/repositories_manager/<name>/authorize`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [repositoriesManagerAuthorizeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+repositoriesManagerAuthorizeHandler%22)
    









### POST `/project/<project-key>/repositories_manager/<name>/authorize/callback`

URL         | **`/project/<project-key>/repositories_manager/<name>/authorize/callback`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [repositoriesManagerAuthorizeCallbackHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+repositoriesManagerAuthorizeCallbackHandler%22)
    









### GET `/project/<project-key>/repositories_manager/<name>/repo`

URL         | **`/project/<project-key>/repositories_manager/<name>/repo`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getRepoFromRepositoriesManagerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getRepoFromRepositoriesManagerHandler%22)
    









### GET `/project/<project-key>/repositories_manager/<name>/repos`

URL         | **`/project/<project-key>/repositories_manager/<name>/repos`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getReposFromRepositoriesManagerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getReposFromRepositoriesManagerHandler%22)
    









### GET `/project/<project-key>/runs`

URL         | **`/project/<project-key>/runs`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowAllRunsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowAllRunsHandler%22)
    









### GET `/project/<project-key>/storage/<integrationName>`

URL         | **`/project/<project-key>/storage/<integrationName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getArtifactsStoreHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getArtifactsStoreHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/artifact/<ref>`

URL         | **`/project/<project-key>/storage/<integrationName>/artifact/<ref>`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobArtifactHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobArtifactHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/artifact/<ref>/url`

URL         | **`/project/<project-key>/storage/<integrationName>/artifact/<ref>/url`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobArtifacWithTempURLHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobArtifacWithTempURLHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/artifact/<ref>/url/callback`

URL         | **`/project/<project-key>/storage/<integrationName>/artifact/<ref>/url/callback`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobArtifactWithTempURLCallbackHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobArtifactWithTempURLCallbackHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/cache/<tag>`

URL         | **`/project/<project-key>/storage/<integrationName>/cache/<tag>`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postPushCacheHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPushCacheHandler%22)
    









### GET `/project/<project-key>/storage/<integrationName>/cache/<tag>`

URL         | **`/project/<project-key>/storage/<integrationName>/cache/<tag>`**
----------- |----------
Method      | GET     
Permissions |  NeedWorker:  -  Auth: true
Code        | [getPullCacheHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPullCacheHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/cache/<tag>/url`

URL         | **`/project/<project-key>/storage/<integrationName>/cache/<tag>/url`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postPushCacheWithTempURLHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPushCacheWithTempURLHandler%22)
    









### GET `/project/<project-key>/storage/<integrationName>/cache/<tag>/url`

URL         | **`/project/<project-key>/storage/<integrationName>/cache/<tag>/url`**
----------- |----------
Method      | GET     
Permissions |  NeedWorker:  -  Auth: true
Code        | [getPullCacheWithTempURLHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPullCacheWithTempURLHandler%22)
    









### POSTEXECUTE `/project/<project-key>/storage/<integrationName>/staticfiles/<name>`

URL         | **`/project/<project-key>/storage/<integrationName>/staticfiles/<name>`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobStaticFilesHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobStaticFilesHandler%22)
    









### GET `/project/<project-key>/variable`

URL         | **`/project/<project-key>/variable`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariablesInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariablesInProjectHandler%22)
    









### PUT `/project/<project-key>/variable/<name>`

URL         | **`/project/<project-key>/variable/<name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateVariableInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateVariableInProjectHandler%22)
    









### GET `/project/<project-key>/variable/<name>`

URL         | **`/project/<project-key>/variable/<name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableInProjectHandler%22)
    









### DELETE `/project/<project-key>/variable/<name>`

URL         | **`/project/<project-key>/variable/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteVariableFromProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteVariableFromProjectHandler%22)
    









### POST `/project/<project-key>/variable/<name>`

URL         | **`/project/<project-key>/variable/<name>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addVariableInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addVariableInProjectHandler%22)
    









### GET `/project/<project-key>/variable/<name>/audit`

URL         | **`/project/<project-key>/variable/<name>/audit`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariableAuditInProjectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariableAuditInProjectHandler%22)
    









### GET `/project/<project-key>/variable/audit`

URL         | **`/project/<project-key>/variable/audit`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getVariablesAuditInProjectnHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getVariablesAuditInProjectnHandler%22)
    









### GET `/project/<project-key>/workflow/<workflow-name>/node/<nodeID>/hook/model`

URL         | **`/project/<project-key>/workflow/<workflow-name>/node/<nodeID>/hook/model`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowHookModelsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowHookModelsHandler%22)
    









### getWorkflows returns ID and name of workflows for a given project/user

URL         | **`/project/<project-key>/workflows`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowsHandler%22)
    









### postWorkflow creates a new workflow

URL         | **`/project/<project-key>/workflows`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowHandler%22)
    









### putWorkflow updates a workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putWorkflowHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putWorkflowHandler%22)
    









### putWorkflow deletes a workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteWorkflowHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteWorkflowHandler%22)
    









### getWorkflow returns a full workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/artifact/<artifactId>`

URL         | **`/project/<project-key>/workflows/<workflow-name>/artifact/<artifactId>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getDownloadArtifactHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getDownloadArtifactHandler%22)
    









### Make the workflow as code

URL         | **`/project/<project-key>/workflows/<workflow-name>/ascode`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowAsCodeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowAsCodeHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/ascode/<uuid>`

URL         | **`/project/<project-key>/workflows/<workflow-name>/ascode/<uuid>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowAsCodeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowAsCodeHandler%22)
    









### postWorkflowGroup add permission for a group on the workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/groups`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowGroupHandler%22)
    









### deleteWorkflowGroup delete permission for a group on the workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/groups/<group-name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteWorkflowGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteWorkflowGroupHandler%22)
    









### putWorkflowGroup update permission for a group on the workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/groups/<group-name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [putWorkflowGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putWorkflowGroupHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/hooks/<uuid>`

URL         | **`/project/<project-key>/workflows/<workflow-name>/hooks/<uuid>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowHookHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowHookHandler%22)
    









### postWorkflowLabel handler to link a label to a workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/label`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowLabelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowLabelHandler%22)
    









### deleteWorkflowLabel handler to unlink a label to a workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/label/<labelID>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteWorkflowLabelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteWorkflowLabelHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/node/<nodeID>/triggers/condition`

URL         | **`/project/<project-key>/workflows/<workflow-name>/node/<nodeID>/triggers/condition`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowTriggerConditionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowTriggerConditionHandler%22)
    









### postWorkflowRollback rollback to a specific audit id

URL         | **`/project/<project-key>/workflows/<workflow-name>/rollback/<auditID>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowRollbackHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowRollbackHandler%22)
    









### POSTEXECUTE `/project/<project-key>/workflows/<workflow-name>/runs`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  AllowServices: true -  Auth: true
Code        | [postWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowRunHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowRunsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowRunsHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>`**
----------- |----------
Method      | GET     
Permissions |  AllowServices: true -  Auth: true
Code        | [getWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowRunHandler%22)
    









### TODO Clean old workflow structure

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/<nodeName>/commits`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowCommitsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowCommitsHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/artifacts`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/artifacts`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowRunArtifactsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowRunArtifactsHandler%22)
    









### POST `/project/<project-key>/workflows/<workflow-name>/runs/<number>/hooks/<hookRunID>/callback`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/hooks/<hookRunID>/callback`**
----------- |----------
Method      | POST     
Permissions |  AllowServices: true -  Auth: true
Code        | [postWorkflowJobHookCallbackHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobHookCallbackHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/hooks/<hookRunID>/details`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/hooks/<hookRunID>/details`**
----------- |----------
Method      | GET     
Permissions |  NeedService:  -  Auth: true
Code        | [getWorkflowJobHookDetailsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowJobHookDetailsHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/<nodeID>/history`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/<nodeID>/history`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowNodeRunHistoryHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowNodeRunHistoryHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowNodeRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowNodeRunHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/info`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/info`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowNodeRunJobSpawnInfosHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowNodeRunJobSpawnInfosHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/log/service`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/log/service`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowNodeRunJobServiceLogsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowNodeRunJobServiceLogsHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/step/<stepOrder>`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/job/<runJobId>/step/<stepOrder>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowNodeRunJobStepHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowNodeRunJobStepHandler%22)
    









### POST `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/release`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/release`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [releaseApplicationWorkflowHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+releaseApplicationWorkflowHandler%22)
    









### POSTEXECUTE `/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/stop`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/nodes/node-run-id/stop`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  Auth: true
Code        | [stopWorkflowNodeRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+stopWorkflowNodeRunHandler%22)
    









### POST `/project/<project-key>/workflows/<workflow-name>/runs/<number>/resync`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/resync`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [resyncWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+resyncWorkflowRunHandler%22)
    









### POSTEXECUTE `/project/<project-key>/workflows/<workflow-name>/runs/<number>/stop`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/stop`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  Auth: true
Code        | [stopWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+stopWorkflowRunHandler%22)
    









### POSTEXECUTE `/project/<project-key>/workflows/<workflow-name>/runs/<number>/vcs/resync`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/<number>/vcs/resync`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  Auth: true
Code        | [postResyncVCSWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postResyncVCSWorkflowRunHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/latest`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/latest`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getLatestWorkflowRunHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getLatestWorkflowRunHandler%22)
    









### postWorkflowRunNum updates the current run number for the given workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/num`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postWorkflowRunNumHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowRunNumHandler%22)
    









### getWorkflowRunNum returns the last run number for the given workflow

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/num`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowRunNumHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowRunNumHandler%22)
    









### GET `/project/<project-key>/workflows/<workflow-name>/runs/tags`

URL         | **`/project/<project-key>/workflows/<workflow-name>/runs/tags`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowRunTagsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowRunTagsHandler%22)
    









