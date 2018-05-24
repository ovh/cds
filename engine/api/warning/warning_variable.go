package warning

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk/log"
)

func variableIsUsed(db gorp.SqlExecutor, key string, varName string) (envsName []string, appsName []string, pipsName []string) {

	// Check if used in environment
	envsName, errE := environment.CountInVarValue(db, key, varName)
	if errE != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in environments: %v", errE)
	}

	// Check if used on application
	appsName, errA := application.CountInVarValue(db, key, varName)
	if errA != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in applications: %v", errA)
	}

	// Check if used on pipeline parameters
	resultsP, errP := pipeline.CountInParamValue(db, key, varName)
	if errP != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in pipeline parameters: %s", errP)
	}

	// Check if used on pipeline jobs
	resultsPip, errP2 := pipeline.CountInPipelines(db, key, varName)
	if errP2 != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in pipelines: %s", errP2)
	}

	pipsName = pipelineSliceMerge(resultsP, resultsPip)
	return
}
