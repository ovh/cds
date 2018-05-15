package warning

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk/log"
)

func variableIsUsed(db gorp.SqlExecutor, key string, varName string) bool {
	used := false

	// Check if used in environment
	resultsE, errE := environment.CountInVarValue(db, key, varName)
	if errE != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in environments: %v", errE)
		return used
	}
	if len(resultsE) > 0 {
		used = true
	}

	// Check if used on application
	resultsA, errA := application.CountInVarValue(db, key, varName)
	if errA != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in applications: %v", errA)
		return used
	}
	if len(resultsA) > 0 {
		used = true
	}

	// Check if used on pipeline parameters
	resultsP, errP := pipeline.CountInParamValue(db, key, varName)
	if errP != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in pipeline parameters: %s", errP)
		return used
	}
	if len(resultsP) > 0 {
		used = true
	}

	// Check if used on pipeline jobs
	resultsPip, errP2 := pipeline.CountInPipelines(db, key, varName)
	if errP2 != nil {
		log.Warning("manageAddVariableEvent> Unable to search variable in pipelines: %s", errP2)
		return used
	}
	if len(resultsPip) > 0 {
		used = true
	}

	return used
}
