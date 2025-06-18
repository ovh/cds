package purge

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	"github.com/ovh/cds/sdk/glob"
	cdslog "github.com/ovh/cds/sdk/log"
)

// WorkflowRunsV2 deletes workflow run v2
func PurgeWorkflowRunsV2(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, purgeRoutineTicker int64) {
	tickPurge := time.NewTicker(time.Duration(purgeRoutineTicker) * time.Hour)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			pkeys, err := project.LoadAllProjectKeys(ctx, DBFunc(), store)
			if err != nil {
				log.Error(ctx, "PurgeWorkflowRunsV2 > unable to list project keys: %v", err)
			}
			for _, pkey := range pkeys {
				ctx := context.WithValue(ctx, cdslog.Project, pkey)
				if err := ApplyRunRetentionOnProject(ctx, DBFunc(), store, pkey); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func ApplyRunRetentionOnProject(ctx context.Context, db *gorp.DbMap, store cache.Store, pkey string) error {
	lockKey := cache.Key("v2", "purge", "run", pkey)
	b, err := store.Lock(lockKey, 5*time.Minute, 100, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer store.Unlock(lockKey)
	log.Info(ctx, "Start PurgeProjectWorkflowRun for project %s", pkey)
	defer log.Info(ctx, "End PurgeProjectWorkflowRun for project %s", pkey)
	projectRunRetention, err := project.LoadRunRetentionByProjectKey(ctx, db, pkey)
	if err != nil {
		return sdk.WrapError(err, "unable to load project run retention")
	}

	// Load workflow
	wnames, err := workflow_v2.LoadRunsWorkflowNames(ctx, db, pkey)
	if err != nil {
		return err
	}
	for _, w := range wnames {
		ApplyRunRetentionOnWorkflow(ctx, db, pkey, w, projectRunRetention)
	}

	return nil
}

func ApplyRunRetentionOnWorkflow(ctx context.Context, db *gorp.DbMap, pkey, workflowFullName string, projectRunRetention *sdk.ProjectRunRetention) {
	nameSplit := strings.Split(workflowFullName, "/")
	if len(nameSplit) != 4 {
		log.Error(ctx, "unable to parse workflow %s. Must be VCS / My / Repo / Workflow", workflowFullName)
		return
	}
	vcs := nameSplit[0]
	repo := nameSplit[1] + "/" + nameSplit[2]
	workflowName := nameSplit[3]
	ctx = context.WithValue(ctx, cdslog.VCSServer, vcs)
	ctx = context.WithValue(ctx, cdslog.Repository, repo)
	ctx = context.WithValue(ctx, cdslog.Workflow, workflowName)

	var workflowRetention *sdk.WorkflowRetentions
	for _, wkfRetention := range projectRunRetention.Retentions.WorkflowRetentions {
		globResult, err := glob.New(wkfRetention.Workflow).MatchString(workflowFullName)
		if err != nil {
			log.Error(ctx, "unable to match glob expression %q with workflow name %q: %v", wkfRetention.Workflow, workflowFullName, err)
			return
		}
		if globResult == nil {
			continue
		}
		workflowRetention = &wkfRetention
		break
	}
	// If no workflow retention found, use the default one
	if workflowRetention == nil {
		workflowRetention = &sdk.WorkflowRetentions{}
	}
	// If not default retention on workflow, retrieve the global one
	if workflowRetention.DefaultRetention == nil {
		workflowRetention.DefaultRetention = &projectRunRetention.Retentions.DefaultRetention
	}

	// Load branches
	refs, err := workflow_v2.LoadRunsWorkflowRefsByWorkflow(ctx, db, pkey, vcs, repo, workflowName)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return
	}
	for _, ref := range refs {
		var ruleRetention *sdk.RetentionRule
		for _, wrr := range workflowRetention.Rules {
			globResult, err := glob.New(wrr.GitRef).MatchString(ref)
			if err != nil {
				log.Error(ctx, "unable to match glob expression %q with ref %q: %v", wrr.GitRef, ref, err)
				return
			}
			if globResult == nil {
				continue
			}
			ruleRetention = &wrr.RetentionRule
			break
		}
		if ruleRetention == nil {
			ruleRetention = workflowRetention.DefaultRetention
		}

		if err := ApplyRunRetentionOnWorkflowRef(ctx, db, pkey, vcs, repo, workflowName, ref, ruleRetention); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}

	}
}

func ApplyRunRetentionOnWorkflowRef(ctx context.Context, db *gorp.DbMap, pkey, vcs, repo, workflowName, ref string, ruleRetention *sdk.RetentionRule) error {
	log.Info(ctx, "Start deleting run for workflow %s/%s/%s/%s on branch %s. Count %d Duration %d", pkey, vcs, repo, workflowName, ref, ruleRetention.Count, ruleRetention.DurationInDays)
	defer log.Info(ctx, "End deleting run for workflow %s/%s/%s/%s on branch %s", pkey, vcs, repo, workflowName, ref)

	// Load old runs
	ids, err := workflow_v2.LoadOlderRuns(ctx, db, pkey, vcs, repo, workflowName, ref, ruleRetention.DurationInDays)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := RemoveWorkflowRunV2(ctx, db, id); err != nil {
			return err
		}
	}

	// Select next run to delete runs
	ids, err = workflow_v2.LoadRunsDescAtOffset(ctx, db, pkey, vcs, repo, workflowName, ref, ruleRetention.Count)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := RemoveWorkflowRunV2(ctx, db, id); err != nil {
			return err
		}
	}

	return nil
}

func RemoveWorkflowRunV2(ctx context.Context, db *gorp.DbMap, id string) error {
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
	log.Info(ctx, "run %s / %s (%d) / %s deleted", run.ProjectKey, run.WorkflowName, run.RunNumber, run.ID)
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
			}
		}
	}

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
	if artifactoryIntegration == nil {
		log.Debug(ctx, "no artifactory integration found")
		return nil
	}

	lowMaturity := artifactoryIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value

	props := utils.NewProperties()
	props.AddProperty("ovh.to_delete", "true")
	props.AddProperty("ovh.to_delete_timestamp", strconv.FormatInt(time.Now().Unix(), 10))

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
