package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func logTime(name string, then time.Time) {
	d := time.Since(then)
	if d > 10*time.Second {
		log.Critical("%s took %s to execute\n", name, d)
		return
	}

	if d > 4*time.Second {
		log.Warning("%s took %s to execute\n", name, d)
		return
	}

	log.Info("%s took %s to execute\n", name, d)
}

//LoadWorkerModelStatusForAdminUser lods worker model status for group
func LoadWorkerModelStatusForAdminUser(db *gorp.DbMap, userID int64) ([]sdk.ModelStatus, error) {
	defer logTime("LoadWorkerModelStatusForAdminUser", time.Now())
	query := `
		SELECT  worker_model.id, 
				worker_model.name, 
				COALESCE(waiting.count, 0) as waiting, 
				COALESCE(building.count,0) as building 
		FROM worker_model
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.model = worker_model.id
				AND (worker.status = $1 OR worker.status = $2)
				GROUP BY model
				) AS waiting ON waiting.model = worker_model.id
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.status = $3
				AND worker.model = worker_model.id
				GROUP BY model
				) AS building ON building.model = worker_model.id
		ORDER BY worker_model.name ASC
		`
	rows, err := db.Query(query, sdk.StatusWaiting.String(), sdk.StatusChecking.String(), sdk.StatusBuilding.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var status []sdk.ModelStatus
	for rows.Next() {
		var ms sdk.ModelStatus
		err := rows.Scan(&ms.ModelID, &ms.ModelName, &ms.CurrentCount, &ms.BuildingCount)
		if err != nil {
			return nil, err
		}
		status = append(status, ms)
	}
	return status, nil
}

//LoadWorkerModelStatusForGroup lods worker model status for group
func LoadWorkerModelStatusForGroup(db *gorp.DbMap, groupID int64) ([]sdk.ModelStatus, error) {
	defer logTime("LoadWorkerModelStatusForGroup", time.Now())

	//Load worker models
	models, errM := LoadWorkerModelsUsableOnGroup(db, groupID, group.SharedInfraGroup.ID)
	if errM != nil {
		return nil, errM
	}
	mapModels := map[int64]sdk.Model{}
	for i, m := range models {
		mapModels[m.ID] = models[i]
	}

	log.Debug("LoadWorkerModelStatusForGroup for group %d, %d", groupID, group.SharedInfraGroup.ID)

	waitingQuery := `SELECT model, COUNT(worker.id) as count FROM worker, worker_model
		WHERE (worker.status = $3 OR worker.status = $4)
		AND (
			worker_model.group_id = $1
			OR 
			worker_model.group_id = $2
			OR
			$1 = $2
		)
		AND worker.model = worker_model.id
		GROUP BY model`

	buildingQuery := `SELECT model, COUNT(worker.id) as count FROM worker, worker_model
		WHERE worker.status = $3
		AND (
			worker_model.group_id = $1
			OR 
			worker_model.group_id = $2
			OR
			$1 = $2
		)
		AND worker.model = worker_model.id
		GROUP BY model`

	type modelCount struct {
		model  int64
		count  int64
		status string
	}

	load := func(c chan modelCount, query string, status string, args ...interface{}) error {
		rows, err := db.Query(query, args...)
		if err != nil {
			log.Warning("LoadWorkerModelStatusForGroup> Error : %s", err)
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var model int64
			var count int64
			if err := rows.Scan(&model, &count); err != nil {
				log.Warning("LoadWorkerModelStatusForGroup> Error : %s", err)
				return err
			}
			log.Debug("[%s] %d %d", status, model, count)
			c <- modelCount{model, count, status}
		}
		return nil
	}

	chanModelCount := make(chan modelCount, 1)
	chanError := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		if err := load(chanModelCount, waitingQuery, "waiting", groupID, group.SharedInfraGroup.ID, sdk.StatusWaiting.String(), sdk.StatusChecking.String()); err != nil {
			chanError <- err
		}
		wg.Done()
	}()
	go func() {
		if err := load(chanModelCount, buildingQuery, "building", groupID, group.SharedInfraGroup.ID, sdk.StatusBuilding.String()); err != nil {
			chanError <- err
		}
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(chanModelCount)
	}()
	go func() {
		err := <-chanError
		log.Critical("LoadWorkerModelStatusForGroup> Error : %s", err)
	}()

	mapModelStatus := map[int64]*sdk.ModelStatus{}

	for _, m := range mapModels {
		ms, ok := mapModelStatus[m.ID]
		//If model status has not been found, load the model

		if !ok || ms == nil {
			mapModelStatus[m.ID] = new(sdk.ModelStatus)
			ms = mapModelStatus[m.ID]
			wm, err := LoadWorkerModelByID(db, m.ID)

			if err != nil {
				log.Warning("LoadWorkerModelStatusForGroup> Unable to load worker model %d", m.ID)
				return nil, err
			}
			ms.ModelID = wm.ID
			ms.ModelGroupID = wm.GroupID
			ms.ModelName = wm.Name
		} else {
			ms.ModelID = m.ID
			ms.ModelGroupID = m.GroupID
			ms.ModelName = m.Name
		}
	}

	for {
		mc, more := <-chanModelCount
		if !more {
			break
		}
		ms, ok := mapModelStatus[mc.model]
		if !ok || ms == nil {
			log.Warning("LoadWorkerModelStatusForGroup> Unable to find model %d in mapModelStatus %v", mc.model, mapModelStatus)
			continue
		}
		if mc.status == "waiting" {
			ms.CurrentCount = mc.count
		} else {
			ms.BuildingCount = mc.count
		}
	}

	var status []sdk.ModelStatus
	for _, v := range mapModelStatus {
		status = append(status, *v)
	}
	return status, nil
}

//ActionCount represents a count of action
type ActionCount struct {
	Action sdk.Action
	Count  int64
}

//LoadGroupActionCount counts waiting action for group
func LoadGroupActionCount(db gorp.SqlExecutor, groupID int64) ([]ActionCount, error) {
	defer logTime("LoadGroupActionCount", time.Now())
	log.Debug("LoadGroupActionCount> Counting pending action for group %d", groupID)
	pbJobs, errJobs := pipeline.GetWaitingPipelineBuildJobForGroup(db, groupID, group.SharedInfraGroup.ID)
	if errJobs != nil {
		if errJobs == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("LoadGroupActionCount> Cannot get waiting pipeline job for group: %s", errJobs)
		return nil, errJobs
	}

	mapAction := map[int64]ActionCount{}
	for _, pbJob := range pbJobs {
		if _, ok := mapAction[pbJob.Job.Action.ID]; ok {
			ac := mapAction[pbJob.Job.Action.ID]
			ac.Count++
			mapAction[pbJob.Job.Action.ID] = ac
		} else {
			mapAction[pbJob.Job.Action.ID] = ActionCount{
				Action: pbJob.Job.Action,
				Count:  1,
			}
		}
	}
	var acs []ActionCount
	for _, value := range mapAction {
		acs = append(acs, value)
	}
	return acs, nil
}

//LoadAllActionCount counts all waiting actions
func LoadAllActionCount(db gorp.SqlExecutor, userID int64) ([]ActionCount, error) {
	defer logTime("LoadAllActionCount", time.Now())
	log.Debug("LoadAllActionCount> Counting pending action")
	pbJobs, errJobs := pipeline.GetWaitingPipelineBuildJob(db)
	if errJobs != nil {
		if errJobs == sql.ErrNoRows {
			return nil, nil
		}
		log.Warning("LoadAllActionCount> Cannot get waiting pipeline job: %s", errJobs)
		return nil, errJobs
	}

	mapAction := map[int64]ActionCount{}
	for _, pbJob := range pbJobs {
		if _, ok := mapAction[pbJob.Job.Action.ID]; ok {
			ac := mapAction[pbJob.Job.Action.ID]
			ac.Count++
			mapAction[pbJob.Job.Action.ID] = ac
		} else {
			mapAction[pbJob.Job.Action.ID] = ActionCount{
				Action: pbJob.Job.Action,
				Count:  1,
			}
		}
	}
	var acs []ActionCount
	for _, value := range mapAction {
		acs = append(acs, value)
	}
	return acs, nil
}

//ModelStatusFunc ...
type ModelStatusFunc func(*gorp.DbMap, int64) ([]sdk.ModelStatus, error)

//ActionCountFunc ...
type ActionCountFunc func(gorp.SqlExecutor, int64) ([]ActionCount, error)

// EstimateWorkerModelNeeds returns for each worker model the needs of instances
func EstimateWorkerModelNeeds(db *gorp.DbMap, uid int64, workerModelStatus ModelStatusFunc, actionCount ActionCountFunc) ([]sdk.ModelStatus, error) {
	defer logTime("EstimateWorkerModelNeeds", time.Now())

	// Load models stats
	ms, errStatus := workerModelStatus(db, uid)
	if errStatus != nil {
		log.Warning("EstimateWorkerModelsNeeds> Cannot LoadWorkerModelStatus  %s\n", errStatus)
		return nil, errStatus
	}

	if log.IsDebug() {
		b, _ := json.Marshal(ms)
		log.Debug("Worker model status : %s ", string(b))
	}

	// Load actions in queue grouped by action (same requirement, same worker model)
	acs, errActionCount := actionCount(db, uid)
	if errActionCount != nil {
		log.Warning("EstimateWorkerModelsNeeds> Cannot LoadActionCount %d: %s\n", uid, errActionCount)
		return nil, errActionCount
	}

	if log.IsDebug() {
		b, _ := json.Marshal(acs)
		log.Debug("Estimate actionCount : %s ", string(b))
	}

	// Now for each unique action in queue, find a worker model able to run it
	for _, ac := range acs {
		// Loop through model in case there is multiple models with the capacity to build current ActionBuild
		// This allow a dispatch of Count via round robin on all matching models
		// Thus dispatching the load potentially on multiple architectures/hatcheries
		loopModels := true
		for loopModels {
			loopModels = false
			for i := range ms {
				capas, errCapa := GetModelCapabilities(db, ms[i].ModelID)
				if errCapa != nil {
					return nil, fmt.Errorf("EstimateWorkerModelNees> cannot GetModelCapabilities: %s\n", errCapa)
				}

				if modelCanRun(db, ms[i].ModelName, ac.Action.Requirements, capas) {
					if ac.Count > 0 {
						ms[i].WantedCount++
						ac.Count--
						loopModels = true
					}

					//Add model requirement if action has specific kind of requirements
					ms[i].Requirements = []sdk.Requirement{}
					for j := range ac.Action.Requirements {
						if ac.Action.Requirements[j].Type == sdk.ServiceRequirement || ac.Action.Requirements[j].Type == sdk.MemoryRequirement {
							ms[i].Requirements = append(ms[i].Requirements, ac.Action.Requirements[j])
						}
					}

					log.Debug("Model %s can run action %d : %d", ms[i].ModelName, ac.Action.ID, ms[i].WantedCount)
				}
			} // !range models
		} // !range loopModels
	} // !range acs

	if log.IsDebug() {
		b, _ := json.Marshal(ms)
		log.Debug("Estimate worker model needs : %s ", string(b))
	}

	return ms, nil
}
