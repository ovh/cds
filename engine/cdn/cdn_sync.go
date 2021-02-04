package cdn

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

const (
	maxWorker    = 5
	cdsSyncQueue = "cdn:cds:sync"
)

var statusSync struct {
	nbProjects          int
	nbProjectsDone      int
	nbProjectsFailed    int
	runPerProjectDone   map[string]int
	runPerProjectFailed map[string]int
	runPerProjectTotal  map[string]int
}

func (s *Service) startCDSSync(ctx context.Context) error {
	return s.Cache.Publish(ctx, cdsSyncQueue, "true")
}

func (s *Service) listenCDSSync(ctx context.Context, cdsStorage *cds.CDS) error {
	pubsub, err := s.Cache.Subscribe(cdsSyncQueue)
	if err != nil {
		return err
	}
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if ctx.Err() != nil {
				continue
			}
			msg, err := pubsub.GetMessage(ctx)
			if err != nil {
				log.Warn(ctx, "cdn.listenCDSSync> cannot get message from pubsub %s: %s", msg, err)
				continue
			}
			if err := s.SyncLogs(ctx, cdsStorage); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
			}
		}
	}
}

// SyncLogs syncs logs from CDS to CDN
func (s *Service) SyncLogs(ctx context.Context, cdsStorage *cds.CDS) error {
	log.Info(ctx, "cdn: Start CDS sync")

	projects, err := cdsStorage.ListProjects()
	if err != nil {
		return err
	}
	statusSync.nbProjects = len(projects)
	statusSync.nbProjectsDone = 0
	statusSync.nbProjectsFailed = 0
	statusSync.runPerProjectDone = make(map[string]int, len(projects))
	statusSync.runPerProjectFailed = make(map[string]int, len(projects))
	statusSync.runPerProjectTotal = make(map[string]int, len(projects))

	log.Info(ctx, "cdn:cds:sync:log: %d projects to sync", len(projects))

	// Browse Project
	for _, p := range projects {
		log.Info(ctx, "cdn:cds:sync:log: project done %d/%d (+%d failed)", statusSync.nbProjectsDone, len(projects), statusSync.nbProjectsFailed)
		if err := s.syncProjectLogs(ctx, cdsStorage, p.Key); err != nil {
			statusSync.nbProjectsFailed++
			log.Error(ctx, "cdn:cds:sync:log  failed to sync project %s: %+v", p.Key, err)
			continue
		}
		statusSync.nbProjectsDone++
	}
	log.Info(ctx, "cdn:cds:sync:log: project done %d/%d (+%d failed)", statusSync.nbProjectsDone, len(projects), statusSync.nbProjectsFailed)
	if statusSync.nbProjectsFailed > 0 {
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

	statusSync.runPerProjectDone[pKey] = 0
	statusSync.runPerProjectFailed[pKey] = 0
	statusSync.runPerProjectTotal[pKey] = 0

	// List of node runs
	nodeRunIdsFromCDS, err := cdsStorage.ListNodeRunIdentifiers(pKey)
	if err != nil {
		return err
	}

	statusSync.runPerProjectTotal[pKey] = len(nodeRunIdsFromCDS)

	// Test if all noderuns have been sync for this project
	listNodeRunsInCDN, err := item.ListNodeRunByProject(s.mustDBWithCtx(ctx), pKey)
	if err != nil {
		return err
	}
	nodeRunMapInCDN := make(map[int64]struct{}, len(listNodeRunsInCDN))
	for _, id := range listNodeRunsInCDN {
		nodeRunMapInCDN[id] = struct{}{}
	}

	nodeRunToMigrate := make([]sdk.WorkflowNodeRunIdentifiers, 0, len(nodeRunIdsFromCDS))
	for _, identifiers := range nodeRunIdsFromCDS {
		if _, has := nodeRunMapInCDN[identifiers.NodeRunID]; !has {
			nodeRunToMigrate = append(nodeRunToMigrate, identifiers)
		}
	}
	statusSync.runPerProjectDone[pKey] = len(nodeRunIdsFromCDS) - len(nodeRunToMigrate)

	log.Info(ctx, "cdn:cds:sync:log: %d node run were already sync for project %s", len(nodeRunIdsFromCDS)-len(nodeRunToMigrate), pKey)
	log.Info(ctx, "cdn:cds:sync:log: %d node run to sync for project %s", len(nodeRunToMigrate), pKey)

	// Nb of nodeRun
	maxNodeRun := len(nodeRunToMigrate)
	jobs := make(chan sdk.WorkflowNodeRunIdentifiers, maxNodeRun)
	results := make(chan error, maxNodeRun)

	// Spawn worker
	for i := 0; i < maxWorker; i++ {
		s.GoRoutines.Exec(ctx, "migrate-noderun-"+strconv.Itoa(i), func(ctx context.Context) {
			s.syncNodeRunJob(ctx, cdsStorage, pKey, jobs, results)
		})
	}

	for i := 0; i < maxNodeRun; i++ {
		jobs <- nodeRunToMigrate[i]
	}
	close(jobs)

	for a := 1; a <= maxNodeRun; a++ {
		err := <-results
		if err != nil {
			statusSync.runPerProjectFailed[pKey]++
			log.Error(ctx, "cdn:cds:sync:log: unable to sync node runs: %v", err)
		} else {
			statusSync.runPerProjectDone[pKey]++
		}
		log.Info(ctx, "cdn:cds:sync:log: node run done for project %s:  %d/%d (+%d failed)", pKey, statusSync.runPerProjectDone[pKey], statusSync.runPerProjectTotal[pKey], statusSync.runPerProjectFailed[pKey])
	}

	if statusSync.runPerProjectFailed[pKey] > 0 {
		return sdk.WithStack(fmt.Errorf("failed during node run sync on project %s", pKey))
	}
	return nil
}

func (s *Service) syncNodeRunJob(ctx context.Context, cdsStorage *cds.CDS, pKey string, jobs <-chan sdk.WorkflowNodeRunIdentifiers, results chan<- error) {
	for j := range jobs {
		ids, err := s.syncNodeRun(ctx, cdsStorage, pKey, j)
		if err == nil {
			for _, itemID := range ids {
				for _, sto := range s.Units.Storages {
					if _, is := sto.(*cds.CDS); is {
						continue
					}
					if err := s.Cache.ScoredSetAdd(ctx, cache.Key(storage.KeyBackendSync, sto.Name()), itemID, float64(time.Now().AddDate(-1, 0, 0).UnixNano())); err != nil {
						log.Info(ctx, "syncNodeRunJob> cannot push item %s into scoredset for unit %s", itemID, sto.Name())
						continue
					}
				}
			}

		}
		results <- err
	}
}

func (s *Service) syncNodeRun(ctx context.Context, cdsStorage *cds.CDS, pKey string, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers) ([]string, error) {
	lockKey := cache.Key("cdn", "log", "sync", strconv.Itoa(int(nodeRunIdentifier.NodeRunID)))
	b, err := s.Cache.Lock(lockKey, 5*time.Minute, 0, 1)
	if err != nil {
		return nil, err
	}
	if !b {
		log.Debug(ctx, "cd:syncNodeRun: already lock %d", nodeRunIdentifier.NodeRunID)
		return nil, nil
	}
	defer s.Cache.Unlock(lockKey) // nolint

	// Load node run
	nodeRun, err := cdsStorage.GetWorkflowNodeRun(pKey, nodeRunIdentifier)
	if err != nil {
		return nil, err
	}

	if !sdk.StatusIsTerminated(nodeRun.Status) {
		return nil, nil
	}

	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	var itemIDs []string
	for _, st := range nodeRun.Stages {
		for _, rj := range st.RunJobs {
			for _, ss := range rj.Job.StepStatus {
				stepName := rj.Job.Action.Actions[ss.StepOrder].StepName
				if stepName == "" {
					stepName = rj.Job.Action.Actions[ss.StepOrder].Name
				}
				id, err := s.syncStepLog(ctx, tx, cdsStorage, pKey, nodeRun, nodeRunIdentifier, rj, ss, stepName)
				if err != nil {
					return nil, err
				}
				itemIDs = append(itemIDs, id)
			}

			dictRequirement := make(map[string]int64)
			for _, r := range rj.Job.Action.Requirements {
				if r.Type == sdk.ServiceRequirement {
					dictRequirement[r.Name] = r.ID
				}
			}
			if len(dictRequirement) > 0 {
				ids, err := s.syncServiceLogs(ctx, tx, cdsStorage, pKey, nodeRun, nodeRunIdentifier, rj, dictRequirement)
				if err != nil {
					return nil, err
				}
				itemIDs = append(itemIDs, ids...)
			}
		}
	}
	return itemIDs, sdk.WithStack(tx.Commit())
}

func (s *Service) syncServiceLogs(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, cdsStorage *cds.CDS, pKey string, nodeRun *sdk.WorkflowNodeRun, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers, rj sdk.WorkflowNodeJobRun, dict map[string]int64) ([]string, error) {
	var itemsIDs []string
	for k, v := range dict {
		apiRef := &sdk.CDNLogAPIRef{
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
		id, err := s.syncItem(ctx, tx, cdsStorage, sdk.CDNTypeItemServiceLog, apiRef)
		if err != nil {
			return nil, err
		}
		itemsIDs = append(itemsIDs, id)
	}
	return itemsIDs, nil
}

func (s *Service) syncStepLog(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, su storage.Interface, pKey string, nodeRun *sdk.WorkflowNodeRun, nodeRunIdentifier sdk.WorkflowNodeRunIdentifiers, rj sdk.WorkflowNodeJobRun, ss sdk.StepStatus, stepName string) (string, error) {
	apiRef := &sdk.CDNLogAPIRef{
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

func (s *Service) syncItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, su storage.Interface, itemType sdk.CDNItemType, apiRef sdk.CDNApiRef) (string, error) {
	apirefHash, err := apiRef.ToHash()
	if err != nil {
		return "", err
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
		return "", nil
	}
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return "", err
	}

	if err := item.Insert(ctx, s.Mapper, tx, it); err != nil {
		return "", err
	}
	// Can't call NewItemUnit because need to complete item first to have hash, to be able to compute locator
	tmpItemUnit := sdk.CDNItemUnit{
		ItemID:       it.ID,
		UnitID:       su.ID(),
		LastModified: time.Now(),
		Item:         it,
	}
	if err := s.completeItem(ctx, tx, tmpItemUnit); err != nil {
		return "", err
	}
	clearItem, err := item.LoadByID(ctx, s.Mapper, tx, it.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return "", err
	}
	itemUnit, err := s.Units.NewItemUnit(ctx, su, clearItem)
	if err != nil {
		return "", err
	}
	if err := storage.InsertItemUnit(ctx, s.Mapper, tx, itemUnit); err != nil {
		return "", err
	}
	return it.ID, nil
}
