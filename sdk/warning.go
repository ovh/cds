package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/ovh/cds/sdk/log"
)

const (
	WarningMissingProjectVariableEnv               = "MISSING_PROJECT_VARIABLE_ENVIRONMENT"
	WarningMissingProjectVariableApplication       = "MISSING_PROJECT_VARIABLE_APPLICATION"
	WarningMissingProjectVariablePipelineParameter = "MISSING_PROJECT_VARIABLE_PIPELINE_PARAMETER"
	WarningMissingProjectVariablePipelineJob       = "MISSING_PROJECT_VARIABLE_PIPELINE_JOB"
	WarningMissingProjectVariableWorkflow          = "MISSING_PROJECT_VARIABLE_WORKFLOW"
	WarningUnusedProjectVariable                   = "UNUSED_PROJECT_VARIABLE"
	WarningMissingProjectPermissionEnv             = "MISSING_PROJECT_PERMISSION_ENV"
	WarningMissingProjectPermissionWorkflow        = "MISSING_PROJECT_PERMISSION_WORKFLOW"
	WarningMissingProjectKeyApplication            = "MISSING_PROJECT_KEY_APPLICATION"
	WarningMissingProjectKeyPipelineParameter      = "MISSING_PROJECT_KEY_PIPELINE_PARAMETER"
	WarningMissingProjectKeyPipelineJob            = "MISSING_PROJECT_KEY_PIPELINE_JOB"
	WarningUnusedProjectKey                        = "UNUSED_PROJECT_KEY"
	WarningMissingProjectVCSServer                 = "MISSING_PROJECT_VCS"
	WarningUnusedProjectVCSServer                  = "UNUSED_PROJECT_VCS"
	WarningMissingVCSConfiguration                 = "MISSING_VCS_CONFIGURATION"
	WarningMissingApplicationVariable              = "MISSING_APPLICATION_VARIABLE"
	WarningUnusedApplicationVariable               = "UNUSED_APPLICATION_VARIABLE"
	WarningMissingApplicationKey                   = "MISSING_APPLICATION_KEY"
	WarningUnusedApplicationKey                    = "UNUSED_APPLICATION_KEY"
	WarningMissingEnvironmentVariable              = "MISSING_ENVIRONMENT_VARIABLE"
	WarningUnusedEnvironmentVariable               = "UNUSED_ENVIRONMENT_VARIABLE"
	WarningMissingEnvironmentKey                   = "MISSING_ENVIRONMENT_KEY"
	WarningUnusedEnvironmentKey                    = "UNUSED_ENVIRONMENT_KEY"
	WarningMissingPipelineParameter                = "MISSING_PIPELINE_PARAMETER"
	WarningUnusedPipelineParameter                 = "UNUSED_PIPELINE_PARAMETER"
)

// WarningV2 Represents warning database structure
type WarningV2 struct {
	ID            int64             `json:"id" db:"id"`
	Key           string            `json:"key" db:"project_key"`
	AppName       string            `json:"application_name" db:"application_name"`
	PipName       string            `json:"pipeline_name" db:"pipeline_name"`
	WorkflowName  string            `json:"workflow_name" db:"workflow_name"`
	EnvName       string            `json:"environment_name" db:"environment_name"`
	Type          string            `json:"type" db:"type"`
	Element       string            `json:"element" db:"element"`
	Created       time.Time         `json:"created" db:"created"`
	MessageParams map[string]string `json:"message_params" db:"-"`
	Message       string            `json:"message" db:"-"`
	Hash          string            `json:"hash" db:"hash"`
}

var MessageAmericanEnglish = map[string]string{
	WarningMissingProjectVariableEnv:               `Variable {{index . "VarName"}} is used by environment {{index . "EnvironmentName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectVariableApplication:       `Variable {{index . "VarName"}} is used by application {{index . "ApplicationName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectVariablePipelineParameter: `Variable {{index . "VarName"}} is used by parameter in pipeline {{index . "PipelineName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectVariablePipelineJob:       `Variable {{index . "VarName"}} is used in pipeline {{index . "PipelineName"}}, in stage {{index . "StageName"}} by job {{index . "JobName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectVariableWorkflow:          `Variable {{index . "VarName"}} is used by the workflow {{index . "WorkflowName"}} in Pipeline {{index . "NodeName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningUnusedProjectVariable:                   `Unused variable {{index . "VarName"}} on project {{index . "ProjectKey"}}.`,
	WarningMissingProjectPermissionEnv:             `Group {{index . "GroupName"}} is not on project {{index . "ProjectKey"}} but is used on Environment {{index . "EnvironmentName"}}.`,
	WarningMissingProjectPermissionWorkflow:        `Group {{index . "GroupName"}} is not on project {{index . "ProjectKey"}} but is used on Workflow {{index . "WorkflowName"}}.`,
	WarningMissingProjectKeyApplication:            `Key {{index . "KeyName"}} is used by application {{index . "ApplicationName"}} but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectKeyPipelineParameter:      `Key {{index . "KeyName"}} is used by a parameter in pipeline {{index . "PipelineName"}}  but does not exist on project {{index . "ProjectKey"}}`,
	WarningMissingProjectKeyPipelineJob:            `Key {{index . "KeyName"}} is used by pipeline {{index . "PipelineName"}} in stage {{index . "StageName"}} in job {{index . "JobName"}}  but does not exist on project {{index . "ProjectKey"}}`,
	WarningUnusedProjectKey:                        `Unused key {{index . "KeyName"}} on project {{index . "ProjectKey"}}.`,
	WarningMissingProjectVCSServer:                 `Repository manager {{index . "VCSName"}} is used by Application: "{{index . "ApplicationName"}}" but does not exist on project {{index . "ProjectKey"}}`,
	WarningUnusedProjectVCSServer:                  `Unused repository manager {{index . "VCSName"}} on project {{index . "ProjectKey"}}.`,
	WarningMissingVCSConfiguration:                 `CDS variables .git.* are used but there is no repository manager on project {{index . "ProjectKey"}}`,
	WarningMissingApplicationVariable:              `Variable {{index . "VarName"}} is used but does not exist on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	WarningUnusedApplicationVariable:               `Unused variable {{index . "VarName"}} on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	WarningMissingApplicationKey:                   `Key {{index . "KeyName"}} is used but does not exist on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	WarningUnusedApplicationKey:                    `Unused key {{index . "KeyName"}} on project/application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	WarningMissingEnvironmentVariable:              `Variable {{index . "VarName"}} is used but does not exist on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	WarningUnusedEnvironmentVariable:               `Unused variable {{index . "VarName"}} on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	WarningMissingEnvironmentKey:                   `Key {{index . "KeyName"}} is used but does not exist on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	WarningUnusedEnvironmentKey:                    `Unused key {{index . "KeyName"}} on project/environment {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	WarningMissingPipelineParameter:                `Parameter {{index . "ParamName"}} is used but does not exist on project/pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}`,
	WarningUnusedPipelineParameter:                 `Unused parameter {{index . "ParamName"}} on project/pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}.`,
}

var MessageFrench = map[string]string{
	WarningMissingProjectVariableEnv:               `La variable {{index . "VarName"}} est utilisée par l'environment {{index . "EnvironmentName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectVariableApplication:       `La variable {{index . "VarName"}} est utilisée par l'application {{index . "ApplicationName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectVariablePipelineParameter: `La variable {{index . "VarName"}} est utilisée par les paramètre du pipeline {{index . "PipelineName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectVariablePipelineJob:       `La variable {{index . "VarName"}} est utilisée par le pipeline {{index . "PipelineName"}}, dans le stage {{index . "StageName"}} dans le job {{index . "JobName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectVariableWorkflow:          `La variable {{index . "VarName"}} est utilisée par le workflow {{index . "WorkflowName"}} par le pipeline {{index . "NodeName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningUnusedProjectVariable:                   `La variable {{index . "VarName"}} est inutilisée dans le projet {{index . "ProjectKey"}}.`,
	WarningMissingProjectPermissionEnv:             `Le groupe {{index . "GroupName"}} n'a pas accès au projet {{index . "ProjectKey"}} mais est positionné sur l'environment {{index . "EnvironmentName"}}.`,
	WarningMissingProjectPermissionWorkflow:        `Le groupe {{index . "GroupName"}} n'a pas accès au projet {{index . "ProjectKey"}} mais est positionné sur le workflow {{index . "WorkflowName"}}.`,
	WarningMissingProjectKeyApplication:            `La clé {{index . "KeyName"}} est utilisée dans l'application {{index . "ApplicationName"}} mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectKeyPipelineParameter:      `La clé {{index . "KeyName"}} est utilisée dans le pipeline {{index . "PipelineName"}}  mais n'existe pas dans le projet {{index . "ProjectKey"}}`,
	WarningMissingProjectKeyPipelineJob:            `La clé {{index . "KeyName"}} est utilisée dans le pipeline {{index . "PipelineName"}} dans le stage {{index . "StageName"}} dans le job {{index . "JobName"}}  but does not exist on project {{index . "ProjectKey"}}`,
	WarningUnusedProjectKey:                        `La clé {{index . "KeyName"}} est inutilisé dans le projet {{index . "ProjectKey"}}.`,
	WarningMissingProjectVCSServer:                 `Le gestionnaire de dépôt {{index . "VCSName"}} est utilisés par l'application' : "{{index . "ApplicationName"}}" mais n'existe pas sur le projet {{index . "ProjectKey"}}`,
	WarningUnusedProjectVCSServer:                  `Le gestionnaire de dépôt {{index . "VCSName"}} est inutilisé dans le project {{index . "ProjectKey"}}`,
	WarningMissingVCSConfiguration:                 `Les variables CDS git.* sont utilisées mais aucun repository manager n'est lié au projet {{index . "ProjectKey"}}`,
	WarningMissingApplicationVariable:              `La variable {{index . "VarName"}} est utilisée mais n'existe pas dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	WarningUnusedApplicationVariable:               `La variable {{index . "VarName"}} est inutilisée dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	WarningMissingApplicationKey:                   `La clé {{index . "KeyName"}} est utilisée mais n'existe pas dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}`,
	WarningUnusedApplicationKey:                    `La clé {{index . "KeyName"}} est inutilisée dans l'application {{index . "ProjectKey"}}/{{index . "ApplicationName"}}.`,
	WarningMissingEnvironmentVariable:              `La variable {{index . "VarName"}} est utilisée mais n'existe pas dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	WarningUnusedEnvironmentVariable:               `La variable {{index . "VarName"}} est inutilisée dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	WarningMissingEnvironmentKey:                   `La clé {{index . "KeyName"}} est utilisée mais n'existe pas dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}`,
	WarningUnusedEnvironmentKey:                    `La clé {{index . "KeyName"}} est inutilisée dans l'environnement {{index . "ProjectKey"}}/{{index . "EnvironmentName"}}.`,
	WarningMissingPipelineParameter:                `Le paramètre {{index . "ParamName"}} est utilisé mais n'existe pas dans le pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}`,
	WarningUnusedPipelineParameter:                 `Le paramètre {{index . "ParamName"}} est inutilisé dans le pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}.`,
}

func (w *WarningV2) ComputeMessage(language string) {
	var buffer bytes.Buffer

	var tmplBody string
	switch language {
	case "fr":
		tmplBody = MessageFrench[w.Type]
	default:
		tmplBody = MessageAmericanEnglish[w.Type]
	}

	// Execute template
	t := template.Must(template.New("warning").Parse(tmplBody))
	if err := t.Execute(&buffer, w.MessageParams); err != nil {
		log.Warning("Unable to compute warning message: %+v: %v", w, err)
		return
	}
	// Set message value
	w.Message = buffer.String()
}

// Warning contains information about user action configuration
type Warning struct {
	ID           int64             `json:"id"`
	Message      string            `json:"message"`
	MessageParam map[string]string `json:"message_param"`

	Action      Action      `json:"action"`
	StageID     int64       `json:"stage_id"`
	Project     Project     `json:"project"`
	Application Application `json:"application"`
	Pipeline    Pipeline    `json:"pipeline"`
	Environment Environment `json:"environment"`
}

// GetWarnings retrieves warnings related to Action accessible to caller
func GetWarnings() ([]Warning, error) {
	uri := "/mon/warning"

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if code > 300 {
		return nil, fmt.Errorf("HTTP %d", code)
	}

	var warnings []Warning
	err = json.Unmarshal(data, &warnings)
	if err != nil {
		return nil, err
	}

	return warnings, nil
}
