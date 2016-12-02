package worker

import (
	"database/sql"
	"fmt"
	"time"

	"encoding/json"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func logTime(name string, then time.Time) {
	d := time.Since(then)
	if d > 10*time.Second {
		log.Critical("%s took %s to execute\n", name, d)
		return
	}

	if d > 2*time.Second {
		log.Warning("%s took %s to execute\n", name, d)
		return
	}

	log.Debug("%s took %s to execute\n", name, d)
}

//LoadWorkerModelStatusForAdminUser lods worker model status for group
func LoadWorkerModelStatusForAdminUser(db *sql.DB, userID int64) ([]sdk.ModelStatus, error) {
	query := `
		SELECT  worker_model.id, 
				worker_model.name, 
				COALESCE(waiting.count, 0) as waiting, 
				COALESCE(building.count,0) as building 
		FROM worker_model
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.status = 'Waiting'
				AND worker.model = worker_model.id
				GROUP BY model
				) AS waiting ON waiting.model = worker_model.id
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.status = 'Building'
				AND worker.model = worker_model.id
				GROUP BY model
				) AS building ON building.model = worker_model.id
		ORDER BY worker_model.name ASC
		`
	rows, err := db.Query(query)
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
func LoadWorkerModelStatusForGroup(db *sql.DB, groupID int64) ([]sdk.ModelStatus, error) {
	//Load SharedInfraGroup
	sharedInfraGroup, errLoad := group.LoadGroup(db, group.SharedInfraGroup)
	if errLoad != nil {
		log.Warning("EstimateWorkerModelsNeeds> Cannot load shared infra group: %s\n", errLoad)
		return nil, errLoad
	}

	log.Debug("LoadWorkerModelStatusForGroup for group %d, %d", groupID, sharedInfraGroup.ID)

	query := `
		SELECT  worker_model.id, 
				worker_model.name, 
				worker_model.group_id,
				COALESCE(waiting.count, 0) as waiting, 
				COALESCE(building.count,0) as building 
		FROM worker_model
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.status = 'Waiting'
				AND (
					worker.group_id = $1
					OR 
					$1 = $2
				)
				AND worker.model = worker_model.id
				GROUP BY model
		) AS waiting ON waiting.model = worker_model.id
		LEFT JOIN LATERAL (
				SELECT model, COUNT(worker.id) as count FROM worker
				WHERE worker.status = 'Building'
				AND (
					worker.group_id = $1
					OR 
					$1 = $2
				)
				AND worker.model = worker_model.id
				GROUP BY model
		) AS building ON building.model = worker_model.id
		WHERE (
			worker_model.group_id IN ($1, $2)
			OR
			$1 = $2
		)
		ORDER BY worker_model.name ASC
		`
	rows, err := db.Query(query, groupID, sharedInfraGroup.ID)
	if err != nil {
		log.Warning("LoadWorkerModelStatusForGroup> Error : %s", err)
		return nil, err
	}
	defer rows.Close()

	var status []sdk.ModelStatus
	for rows.Next() {
		var ms sdk.ModelStatus
		err := rows.Scan(&ms.ModelID, &ms.ModelName, &ms.ModelGroupID, &ms.CurrentCount, &ms.BuildingCount)
		if err != nil {
			log.Warning("LoadWorkerModelStatusForGroup> Error : %s", err)
			return nil, err
		}
		status = append(status, ms)
	}
	return status, nil
}

//ActionCount represents a count of action
type ActionCount struct {
	Action sdk.Action
	Count  int64
}

//LoadGroupActionCount counts waiting action for group
func LoadGroupActionCount(db *sql.DB, groupID int64) ([]ActionCount, error) {

	log.Debug("LoadGroupActionCount> Counting pending action for group %d", groupID)

	acs := []ActionCount{}
	query := `
	SELECT COUNT(action_build.id), pipeline_action.action_id
	FROM action_build
	JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
  	JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
  	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
	WHERE action_build.status = $1 
	AND (
		pipeline_group.group_id = $2
		OR
		(select id from "group" where name = $3) = $2
	)
	GROUP BY pipeline_action.action_id
	LIMIT 1000
	`

	rows, err := db.Query(query, string(sdk.StatusWaiting), groupID, group.SharedInfraGroup)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ac := ActionCount{}
		if err := rows.Scan(&ac.Count, &ac.Action.ID); err != nil {
			return nil, err
		}
		ac.Action.Requirements, err = action.GetRequirements(db, ac.Action.ID)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

//LoadAllActionCount counts all waiting actions
func LoadAllActionCount(db *sql.DB, userID int64) ([]ActionCount, error) {
	acs := []ActionCount{}
	query := `
	SELECT COUNT(action_build.id), pipeline_action.action_id
	FROM action_build
	JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
  	JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
  	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	WHERE action_build.status = $1 
	GROUP BY pipeline_action.action_id
	LIMIT 1000
	`

	rows, err := db.Query(query, string(sdk.StatusWaiting))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ac := ActionCount{}
		if err := rows.Scan(&ac.Count, &ac.Action.ID); err != nil {
			return nil, err
		}
		ac.Action.Requirements, err = action.GetRequirements(db, ac.Action.ID)
		if err != nil {
			return nil, err
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

//ModelStatusFunc ...
type ModelStatusFunc func(*sql.DB, int64) ([]sdk.ModelStatus, error)

//ActionCountFunc ...
type ActionCountFunc func(*sql.DB, int64) ([]ActionCount, error)

// EstimateWorkerModelNeeds returns for each worker model the needs of instances
func EstimateWorkerModelNeeds(db *sql.DB, uid int64, workerModelStatus ModelStatusFunc, actionCount ActionCountFunc) ([]sdk.ModelStatus, error) {
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
		log.Warning("EstimateWorkerModelsNeeds> Cannot LoadActionCount : %s\n", uid, errActionCount)
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
