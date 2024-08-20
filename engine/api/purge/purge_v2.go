package purge

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	cdslog "github.com/ovh/cds/sdk/log"
)

// WorkflowRunsV2 deletes workflow run v2
func WorkflowRunsV2(ctx context.Context, DBFunc func() *gorp.DbMap, purgeRoutineTIcker int64) {
	tickPurge := time.NewTicker(time.Duration(purgeRoutineTIcker) * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			ids, err := workflow_v2.LoadRunIDsToDelete(ctx, DBFunc())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
			for _, id := range ids {
				if err := WorkflowRunV2(ctx, DBFunc(), id); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func WorkflowRunV2(ctx context.Context, db *gorp.DbMap, id string) error {
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return err
	}
	cdnClient := services.NewClient(srvs)

	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	run, err := workflow_v2.LoadAndLockRunByID(ctx, db, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}
	ctx = context.WithValue(ctx, cdslog.Project, run.ProjectKey)
	ctx = context.WithValue(ctx, cdslog.Workflow, run.WorkflowName)

	if err := DeleteArtifactsFromRepositoryManagerV2(ctx, tx, run); err != nil {
		return sdk.WithStack(err)
	}

	_, code, err := cdnClient.DoJSONRequest(ctx, http.MethodPost, "/bulk/item/delete", sdk.CDNMarkDelete{RunV2ID: run.ID}, nil)
	if err != nil || code >= 400 {
		return sdk.WithStack(err)
	}

	if err := workflow_v2.DeleteRunByID(tx, run.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	log.Info(ctx, "run %s / %s / %s deleted", run.ProjectKey, run.WorkflowName, run.ID)
	return nil
}

func DeleteArtifactsFromRepositoryManagerV2(ctx context.Context, db gorp.SqlExecutor, run *sdk.V2WorkflowRun) error {
	proj, err := project.Load(ctx, db, run.ProjectKey, project.LoadOptions.WithClearIntegrations)
	if err != nil {
		return err
	}

	runResults, err := workflow_v2.LoadRunResultsByRunID(ctx, db, run.ID)
	if err != nil {
		return err
	}

	log.Debug(ctx, "found %d results to delete", len(runResults))

	// Prepare artifactClient if available
	// Only one artifact_manager integration is available on a workflow run
	var (
		artifactClient         artifact_manager.ArtifactManager
		artifactoryIntegration *sdk.ProjectIntegration
		rtToken                string
		rtURL                  string
	)

	var integrations []sdk.ProjectIntegration
	for _, integName := range run.WorkflowData.Workflow.Integrations {
		for i := range proj.Integrations {
			if proj.Integrations[i].Name == integName {
				integrations = append(integrations, proj.Integrations[i])
				break
			}
		}
	}
	if artifactoryIntegration == nil {
		return nil
	}

	lowMaturity := artifactoryIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value

	props := utils.NewProperties()
	props.AddProperty("ovh.to_delete", "true")
	props.AddProperty("ovh.to_delete_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	for i := range integrations {
		integ := integrations[i]

		if integ.Model.ArtifactManager {
			rtName := integ.Config[sdk.ArtifactoryConfigPlatform].Value
			rtURL = integ.Config[sdk.ArtifactoryConfigURL].Value
			rtToken = integ.Config[sdk.ArtifactoryConfigToken].Value
			var err error
			artifactClient, err = artifact_manager.NewClient(rtName, rtURL, rtToken)
			if err != nil {
				return err
			}
			artifactoryIntegration = &integ
			break
		}
	}

	for i := range runResults {
		result := &runResults[i]

		// Mark only artifact in snapshot repositories
		if result.ArtifactManagerMetadata.Get("maturity") != lowMaturity {
			continue
		}
		if result.ArtifactManagerIntegrationName == nil {
			continue
		}
		localRepository := result.ArtifactManagerMetadata.Get("localRepository")
		filePath := result.ArtifactManagerMetadata.Get("path")
		fi, err := artifactClient.GetFileInfo(localRepository, filePath)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact info from result %s: %v", result.ID, err)
			continue
		}
		if err := artifactClient.SetProperties(localRepository, fi.Path, props); err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Info(ctx, "unable to mark artifact %q %q (run result %d) to delete: %v", localRepository, fi.Path, result.ID, err)
			continue
		}
	}

	return nil
}
