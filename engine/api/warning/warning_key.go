package warning

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk/log"
)

// keyIsUsed returns if given key is used.
func keyIsUsed(db gorp.SqlExecutor, projectKey string, keyName string) ([]string, []string, []pipeline.CountInPipelineData) {

	// Check if used on application vcs configuration
	resultsApplication, errApp := application.CountApplicationByVcsConfigurationKeys(db, projectKey, keyName)
	if errApp != nil {
		log.Warning("keyIsUsed> Unable to search key in application vcs configuration: %s", errApp)
	}

	// Check if used on pipeline parameters
	resultsPipParam, errP := pipeline.CountInParamValue(db, projectKey, keyName)
	if errP != nil {
		log.Warning("keyIsUsed> Unable to search key in pipeline parameters: %s", errP)
	}

	// Check if used on pipeline jobs
	resultsPip, errP2 := pipeline.CountInPipelines(db, projectKey, keyName)
	if errP2 != nil {
		log.Warning("keyIsUsed> Unable to search key in pipelines: %s", errP2)
	}

	return resultsApplication, resultsPipParam, resultsPip
}
