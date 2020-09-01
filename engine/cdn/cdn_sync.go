package cdn

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) SyncLogs(ctx context.Context, cdsStorage *cds.CDS) error {
	log.Info(ctx, "cdn: Start CDS sync")

	projects, err := cdsStorage.ListProjects()
	if err != nil {
		return err
	}
	log.Info(ctx, "cdn:cds:sync:log: %d projects to sync", len(projects))

	projectDone := 0
	projectFailed := 0
	// Browse Project
	for _, p := range projects {
		log.Info(ctx, "cdn:cds:sync:log: project done %d/%d (+%d failed)", projectDone, len(projects), projectFailed)
		if err := s.syncProjectLogs(ctx, cdsStorage, p.Key); err != nil {
			projectFailed++
			log.Error(ctx, "cdn:cds:sync:log  failed to sync project %s: %+v", p.Key, err)
			continue
		}
		projectDone++
	}
	log.Info(ctx, "cdn:cds:sync:log: project done %d/%d (+%d failed)", projectDone, len(projects), projectFailed)
	if projectFailed > 0 {
		return sdk.WithStack(fmt.Errorf("failures during cds backend sync"))
	}
	return nil
}

func (s *Service) syncProjectLogs(ctx context.Context, cdsStorage *cds.CDS, pKey string) error {
	hasFailed := false
	// Check feature enable
	resp, err := cdsStorage.FeatureEnabled("cdn-job-logs", map[string]string{"project_key": pKey})
	if err != nil {
		return err
	}
	if !resp.Enabled || !s.Cfg.EnableLogProcessing {
		return nil
	}
	// List of node runs
	nodeRunIds, err := cdsStorage.ListNodeRunIdentifiers(pKey)
	if err != nil {
		return err
	}

	nodeRunDone := 0
	nodeRunFailed := 0
	log.Info(ctx, "cdn:cds:sync:log: %d node run to sync for project %s", len(nodeRunIds), pKey)
	// Browse node run
	for _, nodeRunIdentifier := range nodeRunIds {
		log.Info(ctx, "cdn:cds:sync:log: node run done for project %s:  %d/%d (+%d failed)", pKey, nodeRunDone, len(nodeRunIds), nodeRunFailed)
		if err := s.syncNodeRun(ctx, cdsStorage, pKey, nodeRunIdentifier); err != nil {
			hasFailed = true
			nodeRunFailed++
			log.Error(ctx, "cdn:cds:sync:log: unable to sync node runs: %v", err)
			continue
		}
		nodeRunDone++
	}
	log.Info(ctx, "cdn:cds:sync:log: node run done for project %s:  %d/%d (+%d failed)", pKey, nodeRunDone, len(nodeRunIds), nodeRunFailed)
	if hasFailed {
		return sdk.WithStack(fmt.Errorf("failed during node run sync on project %s", pKey))
	}
	return nil
}

func (s *Service) syncNodeRun(ctx context.Context, cdsStorage *cds.CDS, pKey string, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers) error {
	lockKey := cache.Key("cdn", "log", "sync", strconv.Itoa(int(nodeRunIdentifier.NodeRunID)))
	b, err := s.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer s.Cache.Unlock(lockKey)

	// Load node run
	nodeRun, err := cdsStorage.GetWorkflowNodeRun(pKey, nodeRunIdentifier)
	if err != nil {
		return err
	}

	if !sdk.StatusIsTerminated(nodeRun.Status) {
		return nil
	}

	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	for _, st := range nodeRun.Stages {
		for _, rj := range st.RunJobs {
			for _, ss := range rj.Job.StepStatus {
				stepName := rj.Job.Action.Actions[ss.StepOrder].StepName
				if stepName == "" {
					stepName = rj.Job.Action.Actions[ss.StepOrder].Name
				}

				apiRef := index.ApiRef{
					StepOrder:      int64(ss.StepOrder),
					NodeRunID:      nodeRun.ID,
					WorkflowName:   nodeRunIdentifier.WorkflowName,
					WorkflowID:     nodeRunIdentifier.WorkflowID,
					NodeRunJobID:   rj.ID,
					ProjectKey:     pKey,
					RunID:          nodeRunIdentifier.WorkflowRunID,
					StepName:       stepName,
					NodeRunJobName: rj.Job.Action.Name,
					NodeRunName:    nodeRun.WorkflowNodeName,
				}
				apirefHash, err := apiRef.ToHash()
				if err != nil {
					return err
				}
				item := &index.Item{
					Type:       index.TypeItemStepLog,
					ApiRef:     apiRef,
					Status:     index.StatusItemIncoming,
					ApiRefHash: apirefHash,
				}
				// check if item exist
				_, err = index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, apirefHash, index.TypeItemStepLog)
				if err == nil {
					continue
				}
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return err
				}

				if err := index.InsertItem(ctx, s.Mapper, tx, item); err != nil {
					return err
				}
				itemUnit, err := s.Units.NewItemUnit(ctx, s.Mapper, tx, cdsStorage, item)
				if err != nil {
					return err
				}
				if err := storage.InsertItemUnit(ctx, s.Mapper, tx, itemUnit); err != nil {
					return err
				}
				if err := s.completeItem(ctx, tx, *itemUnit); err != nil {
					return err
				}
			}
		}
	}
	return sdk.WithStack(tx.Commit())
}
