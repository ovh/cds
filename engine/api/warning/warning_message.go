package warning

const (
	MissingProjectVariable           = "MISSING_PROJECT_VARIABLE"
	UnusedProjectVariable            = "UNUSED_PROJECT_VARIABLE"
	MissingProjectPermissionEnv      = "MISSING_PROJECT_PERMISSION_ENV"
	MissingProjectPermissionWorkflow = "MISSING_PROJECT_PERMISSION_WORKFLOW"
	MissingProjectKey                = "MISSING_PROJECT_KEY"
	UnusedProjectKey                 = "UNUSED_PROJECT_KEY"
	MissingVCSConfiguration          = "MISSING_VCS_CONFIGURATION"
	MissingApplicationVariable       = "MISSING_APPLICATION_VARIABLE"
	UnusedApplicationVariable        = "UNUSED_APPLICATION_VARIABLE"
	MissingApplicationKey            = "MISSING_APPLICATION_KEY"
	UnusedApplicationKey             = "UNUSED_APPLICATION_KEY"
	MissingEnvironmentVariable       = "MISSING_ENVIRONMENT_VARIABLE"
	UnusedEnvironmentVariable        = "UNUSED_ENVIRONMENT_VARIABLE"
	MissingEnvironmentKey            = "MISSING_ENVIRONMENT_KEY"
	UnusedEnvironmentKey             = "UNUSED_ENVIRONMENT_KEY"
	MissingPipelineParameter         = "MISSING_PIPELINE_PARAMETER"
	UnusedPipelineParameter          = "UNUSED_PIPELINE_PARAMETER"
)

var messageAmericanEnglish = map[string]string{
	MissingProjectVariable:           `Variable {{index . "VarName"}} is used but does not exist on project {{index . "ProjectKey"}}`,
	UnusedProjectVariable:            `Unused variable {{index . "VarName"}} on project {{index . "ProjectKey"}}.`,
	MissingProjectPermissionEnv:      `Group {{index . "GroupName"}} is not on project {{index . "ProjectKey"}} but is used on Environment {{index . "EnvName"}}.`,
	MissingProjectPermissionWorkflow: `Group {{index . "GroupName"}} is not on project {{index . "ProjectKey"}} but is used on Workflow {{index . "WorkflowName"}}.`,
	MissingProjectKey:                `Key {{index . "KeyName"}} is used but does not exist on project {{index . "ProjectKey"}}`,
	UnusedProjectKey:                 `Unused key {{index . "KeyName"}} on project {{index . "ProjectKey"}}.`,
	MissingVCSConfiguration:          `CDS variables .git.* are used but there is no repository manager on project {{index . "ProjectKey"}}`,
	MissingApplicationVariable:       `Variable {{index . "VarName"}} is used but does not exist on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	UnusedApplicationVariable:        `Unused variable {{index . "VarName"}} on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	MissingApplicationKey:            `Key {{index . "KeyName"}} is used but does not exist on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	UnusedApplicationKey:             `Unused key {{index . "KeyName"}} on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	MissingEnvironmentVariable:       `Variable {{index . "VarName"}} is used but does not exist on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	UnusedEnvironmentVariable:        `Unused variable {{index . "VarName"}} on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	MissingEnvironmentKey:            `Key {{index . "KeyName"}} is used but does not exist on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	UnusedEnvironmentKey:             `Unused key {{index . "KeyName"}} on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	MissingPipelineParameter:         `Parameter {{index . "ParamName"}} is used but does not exist on project/pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}`,
	UnusedPipelineParameter:          `Unused parameter {{index . "ParamName"}} on project/pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}.`,
}

var messageFrench = map[string]string{
	MissingProjectVariable:           `La variable de projet {{index . "VarName"}} est utilisée mais n'existe pas dans le projet s{{index . "ProjectKey"}}`,
	UnusedProjectVariable:            `La variable {{index . "VarName"}} est inutilisée dans le projet {{index . "ProjectKey"}}.`,
	MissingProjectPermissionEnv:      `Le groupe {{index . "GroupName"}} n'a pas accès au projet {{index . "ProjectKey"}} mais est positionné sur l'environment {{index . "EnvName"}}.`,
	MissingProjectPermissionWorkflow: `Le groupe {{index . "GroupName"}} n'a pas accès au projet {{index . "ProjectKey"}} mais est positionné sur le workflow {{index . "WorkflowName"}}.`,
	MissingProjectKey:                `La clé {{index . "KeyName"}} est utilisée mais n'existe pas sur le projet {{index . "ProjectKey"}}`,
	UnusedProjectKey:                 `La clé {{index . "KeyName"}} est inutilisé dans le projet {{index . "ProjectKey"}}.`,
	MissingVCSConfiguration:          `Les variables CDS git.* sont utilisées mais aucun repository manager n'est lié au projet {{index . "ProjectKey"}}`,
	MissingApplicationVariable:       `La variable {{index . "VarName"}} est utilisée mais n'existe pas dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	UnusedApplicationVariable:        `La variable {{index . "VarName"}} est inutilisée dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	MissingApplicationKey:            `La clé {{index . "KeyName"}} est utilisée mais n'existe pas dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	UnusedApplicationKey:             `La clé {{index . "KeyName"}} est inutilisée dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	MissingEnvironmentVariable:       `La variable {{index . "VarName"}} est utilisée mais n'existe pas dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	UnusedEnvironmentVariable:        `La variable {{index . "VarName"}} est inutilisée dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	MissingEnvironmentKey:            `La clé {{index . "KeyName"}} est utilisée mais n'existe pas dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	UnusedEnvironmentKey:             `La clé {{index . "KeyName"}} est inutilisée dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	MissingPipelineParameter:         `Le paramètre {{index . "ParamName"}} est utilisé mais n'existe pas dans le pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}`,
	UnusedPipelineParameter:          `Le paramètre {{index . "ParamName"}} est inutilisé dans le pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}.`,
}
