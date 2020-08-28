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
		log.Info(ctx, "cdn:cds:sync:log: project done %d (+%d failed)/%d", projectDone, projectFailed, len(projects))
		if err := s.syncProjectLogs(ctx, cdsStorage, p.Key); err != nil {
			projectFailed++
			log.Error(ctx, "cdn:cds:sync:log  failed to sync project %s: %+v", p.Key, err)
			continue
		}
		projectDone++
	}
	if projectFailed > 0 {
		return sdk.WithStack(fmt.Errorf("failures during cds backend sync"))
	}
	return nil
}

func (s *Service) syncProjectLogs(ctx context.Context, cdsStorage *cds.CDS, pKey string) error {
	// Check feature enable
	resp, err := s.Client.FeatureEnabled("cdn-job-logs", map[string]string{"project_key": pKey})
	if err != nil {
		return err
	}
	if !resp.Enabled {
		return nil
	}
	// List of node runs
	nodeRunIds, err := cdsStorage.ListNodeRunIdentifiers(pKey)
	if err != nil {
		return err
	}

	log.Info(ctx, "cdn:cds:sync:log: %d node run to sync for project %s", len(nodeRunIds), pKey)
	// Browse node run
	for _, nodeRunIdentifier := range nodeRunIds {
		if err := s.syncNodeRun(ctx, cdsStorage, pKey, nodeRunIdentifier); err != nil {
			log.Error(ctx, "cdn:cds:sync:log: unable to sync node runs: %v", err)
			continue
		}
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

	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	// Load node run
	nodeRun, err := cdsStorage.GetWorkflowNodeRun(pKey, nodeRunIdentifier)
	if err != nil {
		return err
	}

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
				if err := index.InsertItem(ctx, s.Mapper, tx, item); err != nil {
					if !sdk.ErrorIs(err, sdk.ErrConflictData) {
						return err
					}
					// If already inserted, continue to next step
					continue
				}

				itemUnit, err := s.Units.NewItemUnit(ctx, s.Mapper, tx, cdsStorage, item)
				if err != nil {
					return err
				}
				if err := storage.InsertItemUnit(ctx, s.Mapper, tx, itemUnit); err != nil {
					return err
				}
				if err := s.completeItem(ctx, *itemUnit); err != nil {
					return err
				}
			}
		}
	}
	return sdk.WithStack(tx.Commit())
}
