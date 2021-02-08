package workflow

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func CheckArtifact(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.WorkflowRun, artifactRef sdk.CDNArtifactAPIRef) (bool, error) {
	if wr.ID != artifactRef.RunID {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload and artifact for this run")
	}
	if sdk.StatusIsTerminated(wr.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated run")
	}

	nodeRunJob, err := LoadNodeJobRun(ctx, db, store, artifactRef.RunJobID)
	if err != nil {
		return false, err
	}
	if sdk.StatusIsTerminated(nodeRunJob.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated job")
	}

	artifactFound := false
loop:
	for _, nodeRuns := range wr.WorkflowNodeRuns {
		if len(nodeRuns) < 1 {
			continue
		}
		// get last noderun
		nodeRun := nodeRuns[0]
		if nodeRun.ID == artifactRef.RunNodeID && sdk.StatusIsTerminated(nodeRun.Status) {
			return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated node run")
		}
		for _, art := range nodeRun.Artifacts {
			if art.Name == artifactRef.ArtifactName {
				artifactFound = true
				break loop
			}
		}
	}

	if err := store.SetWithTTL(cache.Key(), artifactRef, 6*3600); err != nil {
		return false, err
	}

	return !artifactFound, nil
}
