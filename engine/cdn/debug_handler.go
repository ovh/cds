package cdn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// maxDebugItems bounds the number of items reported for a single job. A job has one item per
// step plus its service logs, so this is far above any real job, but the route must not be able
// to return an unbounded payload.
const maxDebugItems = 100

// getDebugJobHandler answers what the CDN knows about the logs of one job: the state of the
// intake keys, the dequeue state of this instance, and the items already stored with their
// storage units. It is strictly read only and never returns any log content.
//
// The job identifier is taken as it is: the intake keys are built the same way for both job
// generations, from the run job id for the recent one and from the numeric job id for the older
// one, so one parameter covers both.
func (s *Service) getDebugJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		jobID := mux.Vars(r)["jobID"]
		if jobID == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing job identifier")
		}

		return service.WriteJSON(w, s.debugJob(ctx, jobID), http.StatusOK)
	}
}

// debugJob gathers the state of a job. Every section reports its own error instead of failing
// the whole answer: an operator looking at an incident needs what could be read, and a single
// unreachable dependency must not hide the rest.
func (s *Service) debugJob(ctx context.Context, jobID string) sdk.CDNDebugJob {
	res := sdk.CDNDebugJob{
		JobID: jobID,
		Items: []sdk.CDNDebugJobItem{},
		Dequeue: sdk.CDNDebugJobDequeue{
			ClaimedQueues: s.dequeuingJobQueues.Load(),
			MaxQueues:     s.maxJobLogsGoroutines(),
		},
	}

	res.Intake, res.Errors = s.debugJobIntake(jobID, res.Errors)
	res.Items, res.ItemsTruncated, res.Errors = s.debugJobItems(ctx, jobID, res.Errors)
	res.Diagnosis = debugJobDiagnosis(res)

	return res
}

// debugJobIntake reads the keys written by the intake for that job.
func (s *Service) debugJobIntake(jobID string, errs []string) (sdk.CDNDebugJobIntake, []string) {
	intake := sdk.CDNDebugJobIntake{
		IncomingQueueKey: cache.Key(keyJobLogQueue, jobID),
		SizeKey:          cache.Key(keyJobLogSize, jobID),
		HeartbeatKey:     cache.Key(keyJobHearbeat, jobID),
		StepMaxSize:      s.Cfg.Log.StepMaxSize,
	}

	if s.Cache == nil {
		return intake, append(errs, "intake: no cache configured on this instance")
	}

	exists, err := s.Cache.Exist(intake.IncomingQueueKey)
	if err != nil {
		errs = append(errs, fmt.Sprintf("intake: unable to check %s: %v", intake.IncomingQueueKey, err))
	} else {
		intake.IncomingQueueExists = exists
	}

	queueLen, err := s.Cache.QueueLen(intake.IncomingQueueKey)
	if err != nil {
		errs = append(errs, fmt.Sprintf("intake: unable to read the length of %s: %v", intake.IncomingQueueKey, err))
	} else {
		intake.IncomingQueueLength = queueLen
	}

	sizeExists, err := s.Cache.Get(intake.SizeKey, &intake.SizeValue)
	if err != nil {
		errs = append(errs, fmt.Sprintf("intake: unable to read %s: %v", intake.SizeKey, err))
	} else {
		intake.SizeExists = sizeExists
	}

	heartbeatExists, err := s.Cache.Get(intake.HeartbeatKey, &intake.HeartbeatOwner)
	if err != nil {
		// The heartbeat used to hold a boolean, so an older value does not read as a string
		errs = append(errs, fmt.Sprintf("intake: unable to read %s: %v", intake.HeartbeatKey, err))
		exists, existErr := s.Cache.Exist(intake.HeartbeatKey)
		if existErr == nil {
			intake.HeartbeatExists = exists
		}
	} else {
		intake.HeartbeatExists = heartbeatExists
	}

	return intake, errs
}

// debugJobItems reports the items stored for that job, with their storage units and, for the
// logs still in the buffer, the number of lines it holds.
func (s *Service) debugJobItems(ctx context.Context, jobID string, errs []string) ([]sdk.CDNDebugJobItem, bool, []string) {
	res := []sdk.CDNDebugJobItem{}

	if s.DBConnectionFactory == nil {
		return res, false, append(errs, "items: no database configured on this instance")
	}
	// Not mustDBWithCtx: it panics when the database is down, which would also throw away the
	// intake section, the one that needs no database at all.
	db := s.DBConnectionFactory.GetDBMap(s.Mapper)()
	if db == nil {
		return res, false, append(errs, "items: database unavailable")
	}
	db = db.WithContext(ctx).(*gorp.DbMap)

	// One more than the bound, to know whether the answer was cut
	items, err := item.LoadByJobIdentifier(ctx, s.Mapper, db, jobID, maxDebugItems+1)
	if err != nil {
		return res, false, append(errs, fmt.Sprintf("items: unable to load the items of the job: %v", err))
	}

	truncated := len(items) > maxDebugItems
	if truncated {
		items = items[:maxDebugItems]
	}

	unitNames := map[string]string{}
	units, err := storage.LoadAllUnits(ctx, s.Mapper, db)
	if err != nil {
		errs = append(errs, fmt.Sprintf("items: unable to load the storage units: %v", err))
	}
	for _, u := range units {
		unitNames[u.ID] = u.Name
	}

	for i := range items {
		debugItem := sdk.CDNDebugJobItem{
			ID:           items[i].ID,
			APIRefHash:   items[i].APIRefHash,
			Type:         items[i].Type,
			Status:       items[i].Status,
			Size:         items[i].Size,
			Created:      items[i].Created,
			LastModified: items[i].LastModified,
			ToDelete:     items[i].ToDelete,
			Units:        []sdk.CDNDebugJobItemUnit{},
		}

		switch apiRef := items[i].APIRef.(type) {
		case *sdk.CDNLogAPIRef:
			debugItem.StepOrder = apiRef.StepOrder
			debugItem.StepName = apiRef.StepName
		case *sdk.CDNLogAPIRefV2:
			debugItem.StepOrder = apiRef.StepOrder
			debugItem.StepName = apiRef.StepName
		}

		itemUnits, err := storage.LoadAllItemUnitsByItemIDIncludingDeleted(ctx, s.Mapper, db, items[i].ID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("items: unable to load the storage units of item %s: %v", items[i].ID, err))
		}
		for j := range itemUnits {
			debugItem.Units = append(debugItem.Units, sdk.CDNDebugJobItemUnit{
				ID:           itemUnits[j].ID,
				UnitID:       itemUnits[j].UnitID,
				UnitName:     unitNames[itemUnits[j].UnitID],
				ToDelete:     itemUnits[j].ToDelete,
				LastModified: itemUnits[j].LastModified,
			})
		}

		if lines, ok := s.debugBufferLines(items[i].Type, itemUnits); ok {
			debugItem.BufferLines = &lines
		}

		res = append(res, debugItem)
	}

	return res, truncated, errs
}

// debugBufferLines returns the number of lines the log buffer holds for an item. It goes
// through the buffer unit rather than rebuilding its key, and reports nothing when the item is
// not a log or is no longer in the buffer.
func (s *Service) debugBufferLines(itemType sdk.CDNItemType, itemUnits []sdk.CDNItemUnit) (int, bool) {
	if !itemType.IsLog() || s.Units == nil {
		return 0, false
	}
	bufferUnit := s.Units.LogsBuffer()
	if bufferUnit == nil {
		return 0, false
	}
	for i := range itemUnits {
		if itemUnits[i].UnitID != bufferUnit.ID() {
			continue
		}
		lines, err := bufferUnit.Card(itemUnits[i])
		if err != nil {
			return 0, false
		}
		return lines, true
	}
	return 0, false
}

// debugJobDiagnosis turns the collected state into plain observations. It states what is there,
// never why: the reader draws the conclusion.
func debugJobDiagnosis(d sdk.CDNDebugJob) []string {
	var res []string

	if len(d.Items) == 0 {
		res = append(res, "no item exists for this job: nothing has ever been stored for it")
	} else {
		var incoming, completed, toDelete int
		for _, it := range d.Items {
			switch it.Status {
			case sdk.CDNStatusItemIncoming:
				incoming++
			case sdk.CDNStatusItemCompleted:
				completed++
			}
			if it.ToDelete {
				toDelete++
			}
		}
		res = append(res, fmt.Sprintf("%d item(s) found: %d completed, %d incoming", len(d.Items), completed, incoming))
		if toDelete > 0 {
			res = append(res, fmt.Sprintf("%d item(s) are marked for deletion", toDelete))
		}
	}
	if d.ItemsTruncated {
		res = append(res, fmt.Sprintf("the item list is truncated at %d", maxDebugItems))
	}

	if d.Intake.SizeExists && d.Intake.StepMaxSize > 0 && d.Intake.SizeValue >= d.Intake.StepMaxSize {
		res = append(res, fmt.Sprintf("the size counter is at or above the step max size (%d >= %d): further non terminated lines of this job are dropped by the intake", d.Intake.SizeValue, d.Intake.StepMaxSize))
	}

	if d.Intake.IncomingQueueLength > 0 {
		res = append(res, fmt.Sprintf("%d message(s) are still waiting in the incoming queue", d.Intake.IncomingQueueLength))
	}

	switch {
	case d.Intake.HeartbeatExists && d.Intake.HeartbeatOwner != "":
		res = append(res, fmt.Sprintf("the incoming queue is claimed by %s", d.Intake.HeartbeatOwner))
	case d.Intake.HeartbeatExists:
		res = append(res, "the incoming queue is claimed, by an instance that did not record its identifier")
	case d.Intake.IncomingQueueLength > 0:
		res = append(res, "the incoming queue is not claimed by any instance")
	}

	if d.Dequeue.MaxQueues > 0 && d.Dequeue.ClaimedQueues >= d.Dequeue.MaxQueues {
		res = append(res, fmt.Sprintf("the instance that answered is at its dequeue cap (%d/%d)", d.Dequeue.ClaimedQueues, d.Dequeue.MaxQueues))
	}

	return res
}
