package warning

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk/log"
)

// keyIsUsed returns if given key is used.
// Return []string (applications name), []string (pipelines name)
func keyIsUsed(db gorp.SqlExecutor, projectKey string, keyName string) ([]string, []string) {

	// Check if used on application vcs configuration
	resultsApplication, errApp := application.CountKeysInVcsConfiguration(db, projectKey, keyName)
	if errApp != nil {
		log.Warning("keyIsUsed> Unable to search key in application vcs configuration: %s", errApp)
	}

	// Check if used on pipeline parameters
	resultsP, errP := pipeline.CountInParamValue(db, projectKey, keyName)
	if errP != nil {
		log.Warning("keyIsUsed> Unable to search key in pipeline parameters: %s", errP)
	}

	// Check if used on pipeline jobs
	resultsPip, errP2 := pipeline.CountInPipelines(db, projectKey, keyName)
	if errP2 != nil {
		log.Warning("keyIsUsed> Unable to search key in pipelines: %s", errP2)
	}

	elementsMap := make(map[string]bool)
	pipelinesName := make([]string, 0)
	for _, v := range resultsP {
		if !elementsMap[v.Name] {
			elementsMap[v.Name] = true
			pipelinesName = append(pipelinesName, v.Name)
		}
	}
	for _, v := range resultsPip {
		if !elementsMap[v.PipName] {
			elementsMap[v.PipName] = true
			pipelinesName = append(pipelinesName, v.PipName)
		}
	}

	return resultsApplication, pipelinesName
}
