package warning

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var projVarRegexp = regexp.MustCompile(`cds\.proj\.[a-zA-Z0-9\-_]+`)

func checkContentValueToAddUnusedWarning(ctx context.Context, db gorp.SqlExecutor, projKey string, varValue string, varPrefix string, reg *regexp.Regexp, warName string) error {
	// check if value contains a project variable
	if strings.Contains(varValue, varPrefix) {
		variables := reg.FindAllString(varValue, -1)
		for _, v := range variables {
			switch varPrefix {
			case "cds.proj":
				if err := checkUnusedProjectVariable(ctx, db, projKey, v, warName); err != nil {
					return sdk.WrapError(err, "Unable to check porject var for unused warning")
				}
			}
		}
	}
	return nil
}

func checkContentValueToRemoveUnusedWarning(ctx context.Context, db gorp.SqlExecutor, projKey string, varValue string, varPrefix string, reg *regexp.Regexp, warName string) error {
	// check if value contains a project variable
	if strings.Contains(varValue, varPrefix) {
		// extract all project var
		variables := reg.FindAllString(varValue, -1)
		for _, v := range variables {
			if err := removeProjectWarning(ctx, db, warName, v, projKey); err != nil {
				return sdk.WrapError(err, "Unable to remove warning from %s", warName)
			}
		}
	}
	return nil
}

func checkUnusedProjectVariable(ctx context.Context, db gorp.SqlExecutor, projectKey string, varName string, warnName string) error {
	ws, envs, apps, pips, pipJobs := variableIsUsed(ctx, db, projectKey, varName)
	if len(ws) == 0 && len(envs) == 0 && len(apps) == 0 && len(pips) == 0 && len(pipJobs) == 0 {
		w := sdk.Warning{
			Key:     projectKey,
			Element: varName,
			Created: time.Now(),
			Type:    warnName,
			MessageParams: map[string]string{
				"VarName":    varName,
				"ProjectKey": projectKey,
			},
		}
		if err := Insert(ctx, db, w); err != nil {
			return sdk.WrapError(err, "Unable to Insert warning")
		}
	}
	return nil
}

func variableIsUsed(ctx context.Context, db gorp.SqlExecutor, key string, varName string) ([]workflow.CountVarInWorkflowData, []string, []string, []string, []pipeline.CountInPipelineData) {
	ws, errWS := workflow.CountVariableInWorkflow(db, key, varName)
	if errWS != nil {
		log.Warning(ctx, "manageAddVariableEvent> Unable to search variable in workflow: %v", errWS)
	}

	// Check if used in environment
	envsName, errE := environment.CountEnvironmentByVarValue(db, key, varName)
	if errE != nil {
		log.Warning(ctx, "manageAddVariableEvent> Unable to search variable in environments: %v", errE)
	}

	// Check if used on application
	appsName, errA := application.CountInVarValue(db, key, varName)
	if errA != nil {
		log.Warning(ctx, "manageAddVariableEvent> Unable to search variable in applications: %v", errA)
	}

	// Check if used on pipeline parameters
	pipsName, errP := pipeline.CountInParamValue(db, key, varName)
	if errP != nil {
		log.Warning(ctx, "manageAddVariableEvent> Unable to search variable in pipeline parameters: %s", errP)
	}

	// Check if used on pipeline jobs
	pipsJob, errP2 := pipeline.CountInPipelines(db, key, varName)
	if errP2 != nil {
		log.Warning(ctx, "manageAddVariableEvent> Unable to search variable in pipelines: %s", errP2)
	}

	return ws, envsName, appsName, pipsName, pipsJob
}
