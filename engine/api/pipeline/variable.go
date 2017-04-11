package pipeline

import (
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ProcessPipelineBuildVariables gathers together parameters from:
// - RunRequest -> add "cds.pip." prefix to them
// - Trigger parameters
// - Default PipelineApplication parameters
// - Builtin cds variables (.cds.* and .git.*)
func ProcessPipelineBuildVariables(pipelineParams []sdk.Parameter, applicationPipelineArgs []sdk.Parameter, buildArgs []sdk.Parameter) (map[string]sdk.Parameter, error) {
	abv := make(map[string]sdk.Parameter)
	pipeline := "cds.pip"

	// Add pipeline parameters and add prefix
	for _, p := range pipelineParams {
		p.Name = pipeline + "." + p.Name
		abv[p.Name] = p
	}

	// Add default pipeline parameters for given application
	for _, p := range applicationPipelineArgs {
		p.Name = pipeline + "." + p.Name
		abv[p.Name] = p
	}

	// Add builtin CDS variables
	for _, p := range buildArgs {
		if !strings.HasPrefix(p.Name, "cds.") && !strings.HasPrefix(p.Name, "git.") {
			log.Debug("ProcessPipelineBuildVariables> Renaming %s into %s\n", p.Name, pipeline+"."+p.Name)
			p.Name = pipeline + "." + p.Name
		}
		abv[p.Name] = p
	}

	return abv, nil
}
