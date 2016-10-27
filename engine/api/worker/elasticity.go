package worker

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	modelCapabilities       map[int64][]sdk.Requirement
	modelCapaMutex          sync.RWMutex
	actionRequirements      map[int64][]sdk.Requirement
	actionRequirementsMutex sync.RWMutex
)

func logTime(name string, then time.Time) {
	d := time.Since(then)
	if d > 10*time.Second {
		log.Warning("%s took %s to execute\n", name, d)
		return
	}

	if d > 2*time.Second {
		log.Warning("%s took %s to execute\n", name, d)
		return
	}

	log.Debug("%s took %s to execute\n", name, d)
}

func loadWorkerModelStatus(db *sql.DB, c *context.Context) ([]sdk.ModelStatus, error) {
	defer logTime("loadWorkerModelStatus", time.Now())

	switch c.Agent {
	case sdk.HatcheryAgent:
		return loadWorkerModelStatusForGroup(db, c.User.Groups[0].ID)
	default:
		return loadWorkerModelStatusForUser(db, c.User.ID)
	}
}

func loadWorkerModelStatusForGroup(db *sql.DB, groupID int64) ([]sdk.ModelStatus, error) {
	query := `
SELECT worker_model.id, worker_model.name, COALESCE(waiting.count, 0) as waiting, COALESCE(building.count,0) as building FROM worker_model
	LEFT JOIN LATERAL (SELECT model, COUNT(worker.id) as count FROM worker
		WHERE worker.group_id = $1 AND worker.status = 'Waiting'
		AND worker.model = worker_model.id
		GROUP BY model) AS waiting ON waiting.model = worker_model.id
	LEFT JOIN LATERAL (SELECT model, COUNT(worker.id) as count FROM worker
		WHERE worker.group_id = $1 AND worker.status = 'Building'
		AND worker.model = worker_model.id
		GROUP BY model) AS building ON building.model = worker_model.id
ORDER BY worker_model.name ASC;
`
	rows, err := db.Query(query, groupID)
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

// loadWorkerModelStatus loads from database the number of worker deployed for each model
// start with group permissions calling user has access to.
func loadWorkerModelStatusForUser(db *sql.DB, userID int64) ([]sdk.ModelStatus, error) {

	query := `
SELECT worker_model.id, worker_model.name, COALESCE(waiting.count, 0) as waiting, COALESCE(building.count,0) as building FROM worker_model
	LEFT JOIN LATERAL (SELECT model, COUNT(worker.id) as count FROM worker
		JOIN "group" ON "group".id = worker.group_id
		JOIN group_user ON "group".id = group_user.group_id
		WHERE group_user.user_id = $1 AND worker.status = 'Waiting'
		AND worker.model = worker_model.id
		GROUP BY model) AS waiting ON waiting.model = worker_model.id
	LEFT JOIN LATERAL (SELECT model, COUNT(worker.id) as count FROM worker
		JOIN "group" ON "group".id = worker.group_id
		JOIN group_user ON "group".id = group_user.group_id
		WHERE group_user.user_id = $1 AND worker.status = 'Building'
		AND worker.model = worker_model.id
		GROUP BY model) AS building ON building.model = worker_model.id
ORDER BY worker_model.name ASC;
`

	rows, err := db.Query(query, userID)
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

func modelCanRun(db *sql.DB, name string, req []sdk.Requirement, capa []sdk.Requirement) bool {
	defer logTime("compareRequirements", time.Now())

	m, err := LoadWorkerModel(db, name)
	if err != nil {
		log.Warning("modelCanRun> Unable to load model %s", name)
		return false
	}

	log.Info("Comparing %d requirements to %d capa\n", len(req), len(capa))
	for _, r := range req {
		// service requirement are only supported by docker model
		if r.Type == sdk.ServiceRequirement && m.Type != sdk.Docker {
			return false
		}

		found := false

		// If requirement is a Model requirement, it's easy. It's either can or can't run
		if r.Type == sdk.ModelRequirement {
			return r.Value == name
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement {
			return false // TODO: update when hatchery in local mode declare an hostname capa
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement {
			continue
		}

		// Everyone can play plugins
		if r.Type == sdk.PluginRequirement {
			continue
		}

		// Check binary requirement against worker model capabilities
		for _, c := range capa {
			log.Debug("Comparing [%s] and [%s]\n", r.Name, c.Name)
			if r.Value == c.Value || r.Value == c.Name {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

type actioncount struct {
	Action sdk.Action
	Count  int64
}

func scanActionCount(db *sql.DB, s database.Scanner) (actioncount, error) {
	ac := actioncount{}
	var actionID int64

	err := s.Scan(&ac.Count, &actionID)
	if err != nil {
		return ac, fmt.Errorf("scanActionCount> cannot scan: %s", err)
	}

	actionRequirementsMutex.RLock()
	req, ok := actionRequirements[actionID]
	actionRequirementsMutex.RUnlock()
	if !ok {
		req, err = action.LoadActionRequirements(db, actionID)
		if err != nil {
			return ac, fmt.Errorf("scanActionCount> cannot LoadActionRequirements for %d: %s\n", actionID, err)
		}
		actionRequirementsMutex.Lock()
		actionRequirements[actionID] = req
		actionRequirementsMutex.Unlock()
	}

	log.Debug("Action %d: %d in queue with %d requirements\n", actionID, ac.Count, len(req))
	ac.Action.Requirements = req
	return ac, nil
}

func loadGroupActionCount(db *sql.DB, groupID int64) ([]actioncount, error) {
	acs := []actioncount{}
	query := `
	SELECT COUNT(action_build.id), pipeline_action.action_id
	FROM action_build
	JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
  JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
  JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
	WHERE action_build.status = $1 AND pipeline_group.group_id = $2
	GROUP BY pipeline_action.action_id
	LIMIT 1000
	`

	rows, err := db.Query(query, string(sdk.StatusWaiting), groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ac, err := scanActionCount(db, rows)
		if err != nil {
			return nil, err
		}

		acs = append(acs, ac)
	}

	return acs, nil
}
func loadUserActionCount(db *sql.DB, userID int64) ([]actioncount, error) {
	acs := []actioncount{}
	query := `
	SELECT COUNT(action_build.id), pipeline_action.action_id
	FROM action_build
	JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
  JOIN pipeline_build ON pipeline_build.id = action_build.pipeline_build_id
  JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
	JOIN group_user ON group_user.group_id = pipeline_group.group_id
	WHERE action_build.status = $1 AND group_user.user_id = $2
	GROUP BY pipeline_action.action_id
	LIMIT 1000
	`

	rows, err := db.Query(query, string(sdk.StatusWaiting), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ac, err := scanActionCount(db, rows)
		if err != nil {
			return nil, err
		}

		acs = append(acs, ac)
	}

	return acs, nil
}

func loadActionCount(db *sql.DB, c *context.Context) ([]actioncount, error) {
	defer logTime("EstimateWorkerModelNeeds", time.Now())

	switch c.Agent {
	case sdk.HatcheryAgent:
		return loadGroupActionCount(db, c.User.Groups[0].ID)
	default:
		return loadUserActionCount(db, c.User.ID)
	}

}

// UpdateModelCapabilitiesCache updates model capabilities cache
func UpdateModelCapabilitiesCache() {
	modelCapabilities = make(map[int64][]sdk.Requirement)

	for {
		time.Sleep(5 * time.Second)
		db := database.DB()
		if db != nil {
			wms, err := LoadWorkerModels(db)
			if err != nil {
				log.Warning("updateModelCapabilities> Cannot load worker models: %s\n", err)
			}
			modelCapaMutex.Lock()
			for _, wm := range wms {
				modelCapabilities[wm.ID] = wm.Capabilities
			}
			modelCapaMutex.Unlock()
		}
	}
}

// UpdateActionRequirementsCache updates internal action cache
func UpdateActionRequirementsCache() {
	actionRequirements = make(map[int64][]sdk.Requirement)

	for {
		time.Sleep(10 * time.Second)
		db := database.DB()
		if db != nil {
			actionRequirementsMutex.Lock()
			for actionID := range actionRequirements {
				req, err := action.LoadActionRequirements(db, actionID)
				if err != nil {
					log.Warning("UpdateActionRequirementsCache> Cannot LoadActionRequirements for %d: %s\n", actionID, err)
					continue
				}
				actionRequirements[actionID] = req
			}
			actionRequirementsMutex.Unlock()
		}
	}
}

// EstimateWorkerModelNeeds returns for each worker model the needs of instances
func EstimateWorkerModelNeeds(db *sql.DB, c *context.Context) ([]sdk.ModelStatus, error) {
	defer logTime("EstimateWorkerModelNeeds", time.Now())
	u := c.User

	// Load models stats
	ms, err := loadWorkerModelStatus(db, c)
	if err != nil {
		log.Warning("EstimateWorkerModelsNeeds> Cannot LoadWorkerModelStatus for user %d: %s\n", u.ID, err)
		return nil, fmt.Errorf("EstimateWorkerModelNeeds> Cannot loadWorkerModelStatus> %s", err)
	}

	// Load actions in queue grouped by action (same requirement, same worker model)
	acs, err := loadActionCount(db, c)
	if err != nil {
		log.Warning("EstimateWorkerModelsNeeds> Cannot LoadActionCount for user %d: %s\n", u.ID, err)
		return nil, fmt.Errorf("EstimateWorkerModelNeeds> cannot loadActionCount> %s", err)
	}

	// Now for each unique action in queue, find a worker model able to run it
	var capas []sdk.Requirement
	var ok bool
	for _, ac := range acs {
		// Loop through model in case there is multiple models with the capacity to build current ActionBuild
		// This allow a dispatch of Count via round robin on all matching models
		// Thus dispatching the load potentially on multiple architectures/hatcheries
		loopModels := true
		for loopModels {
			loopModels = false

			for i := range ms {
				modelCapaMutex.RLock()
				capas, ok = modelCapabilities[ms[i].ModelID]
				modelCapaMutex.RUnlock()
				if !ok {
					capas, err = LoadWorkerModelCapabilities(db, ms[i].ModelID)
					if err != nil {
						return nil, fmt.Errorf("EstimateWorkerModelNees> cannot loadWorkerModelCapabilities: %s\n", err)
					}
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
						if ac.Action.Requirements[j].Type == sdk.ServiceRequirement {
							ms[i].Requirements = append(ms[i].Requirements, ac.Action.Requirements[j])
						}
					}

					//break
				}
			} // !range models
		} // !range loopModels
	} // !range acs

	return ms, nil
}
