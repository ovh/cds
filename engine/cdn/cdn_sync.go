package cdn

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	maxWorker = 5
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

	// Nb of nodeRun
	maxNodeRun := len(nodeRunIds)
	jobs := make(chan sdk.WorkflowNodeRunIdentifiers, maxNodeRun)
	results := make(chan error, maxNodeRun)

	// Spawn worker
	for i := 0; i < maxWorker; i++ {
		sdk.GoRoutine(ctx, "migrate-noderun-"+strconv.Itoa(i), func(ctx context.Context) {
			s.syncNodeRunJob(ctx, cdsStorage, pKey, jobs, results)
		})
	}

	for i := 0; i < len(nodeRunIds); i++ {
		jobs <- nodeRunIds[i]
	}
	close(jobs)

	for a := 1; a <= len(nodeRunIds); a++ {
		err := <-results
		if err != nil {
			nodeRunFailed++
			log.Error(ctx, "cdn:cds:sync:log: unable to sync node runs: %v", err)
		} else {
			nodeRunDone++
		}
		log.Info(ctx, "cdn:cds:sync:log: node run done for project %s:  %d/%d (+%d failed)", pKey, nodeRunDone, len(nodeRunIds), nodeRunFailed)
	}

	if nodeRunFailed > 0 {
		return sdk.WithStack(fmt.Errorf("failed during node run sync on project %s", pKey))
	}
	return nil
}

func (s *Service) syncNodeRunJob(ctx context.Context, cdsStorage *cds.CDS, pKey string, jobs <-chan sdk.WorkflowNodeRunIdentifiers, results chan<- error) {
	for j := range jobs {
		results <- s.syncNodeRun(ctx, cdsStorage, pKey, j)
	}
}

func (s *Service) syncNodeRun(ctx context.Context, cdsStorage *cds.CDS, pKey string, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers) error {
	lockKey := cache.Key("cdn", "log", "sync", strconv.Itoa(int(nodeRunIdentifier.NodeRunID)))
	b, err := s.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		return err
	}
	if !b {
		log.Debug("cd:syncNodeRun: already lock %d", nodeRunIdentifier.NodeRunID)
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
	defer tx.Rollback() // nolint

	for _, st := range nodeRun.Stages {
		for _, rj := range st.RunJobs {
			for _, ss := range rj.Job.StepStatus {
				stepName := rj.Job.Action.Actions[ss.StepOrder].StepName
				if stepName == "" {
					stepName = rj.Job.Action.Actions[ss.StepOrder].Name
				}
				if err := s.syncStepLog(ctx, tx, cdsStorage, pKey, nodeRun, nodeRunIdentifier, rj, ss, stepName); err != nil {
					return err
				}
			}

			dictRequirement := make(map[string]int64)
			for _, r := range rj.Job.Action.Requirements {
				if r.Type == sdk.ServiceRequirement {
					dictRequirement[r.Name] = r.ID
				}
			}
			if len(dictRequirement) > 0 {
				if err := s.syncServiceLogs(ctx, tx, cdsStorage, pKey, nodeRun, nodeRunIdentifier, rj, dictRequirement); err != nil {
					return err
				}
			}

		}
	}
	return sdk.WithStack(tx.Commit())
}

func (s *Service) syncServiceLogs(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, cdsStorage *cds.CDS, pKey string, nodeRun *sdk.WorkflowNodeRun, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers, rj sdk.WorkflowNodeJobRun, dict map[string]int64) error {
	for k, v := range dict {
		apiRef := sdk.CDNLogAPIRef{
			NodeRunID:              nodeRun.ID,
			WorkflowName:           nodeRunIdentifier.WorkflowName,
			WorkflowID:             nodeRunIdentifier.WorkflowID,
			NodeRunJobID:           rj.ID,
			ProjectKey:             pKey,
			RunID:                  nodeRunIdentifier.WorkflowRunID,
			NodeRunJobName:         rj.Job.Action.Name,
			NodeRunName:            nodeRun.WorkflowNodeName,
			RequirementServiceName: k,
			RequirementServiceID:   v,
		}
		if err := s.syncItem(ctx, tx, cdsStorage, sdk.CDNTypeItemServiceLog, apiRef); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncStepLog(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, su storage.Interface, pKey string, nodeRun *sdk.WorkflowNodeRun, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers, rj sdk.WorkflowNodeJobRun, ss sdk.StepStatus, stepName string) error {
	apiRef := sdk.CDNLogAPIRef{
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
	return s.syncItem(ctx, tx, su, sdk.CDNTypeItemStepLog, apiRef)
}

func (s *Service) syncItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, su storage.Interface, itemType sdk.CDNItemType, apiRef sdk.CDNLogAPIRef) error {
	apirefHash, err := apiRef.ToHash()
	if err != nil {
		return err
	}
	it := &sdk.CDNItem{
		Type:       itemType,
		APIRef:     apiRef,
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: apirefHash,
	}
	// check if item exist
	_, err = item.LoadByAPIRefHashAndType(ctx, s.Mapper, tx, apirefHash, itemType)
	if err == nil {
		return nil
	}
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	if err := item.Insert(ctx, s.Mapper, tx, it); err != nil {
		return err
	}
	// Can't call NewItemUnit because need to complete item first to have hash, to be able to compute locator
	tmpItemUnit := sdk.CDNItemUnit{
		ItemID:       it.ID,
		UnitID:       su.ID(),
		LastModified: time.Now(),
		Item:         it,
	}
	if err := s.completeItem(ctx, tx, tmpItemUnit); err != nil {
		return err
	}
	clearItem, err := item.LoadByID(ctx, s.Mapper, tx, it.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}
	itemUnit, err := s.Units.NewItemUnit(ctx, su, clearItem)
	if err != nil {
		return err
	}
	if err := storage.InsertItemUnit(ctx, s.Mapper, tx, itemUnit); err != nil {
		return err
	}
	return nil
}
