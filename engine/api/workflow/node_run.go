package workflow

import (
	"github.com/ovh/cds/sdk"
)

func MergeArtifactWithPreviousSubRun(runs []sdk.WorkflowNodeRun) []sdk.WorkflowNodeRunArtifact {
	switch len(runs) {
	case 0:
		return nil
	case 1:
		return runs[0].Artifacts
	default:
		artifacts := runs[0].Artifacts
		// Only browse subnumber if noderun is still building and has been launched by OnlyFailedJobs option
		if runs[0].Manual != nil && runs[0].Manual.OnlyFailedJobs {

			// Create a map to identify artifacts already get
			tmpsArtifactsMap := make(map[string]struct{})
			for _, art := range artifacts {
				tmpsArtifactsMap[art.Name] = struct{}{}
			}

			// Browse previous subnumber to get list of artifact to get
			for i := 1; i < len(runs); i++ {
				previousRun := runs[i]
				for _, art := range previousRun.Artifacts {
					// If the artifacts has been reupload by a more recent node run, ignored it
					if _, ok := tmpsArtifactsMap[art.Name]; ok {
						continue
					}
					tmpsArtifactsMap[art.Name] = struct{}{}
					artifacts = append(artifacts, art)
				}

				// Stop browsing, if current subnumber has not been launch with OnlyFailJobs option
				if previousRun.Manual == nil || !previousRun.Manual.OnlyFailedJobs {
					break
				}
			}
		}
		return artifacts
	}
}
