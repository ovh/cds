package sanity

import (
	//"database/sql"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// CheckActionRequirements checks:
// - requirements are compabtible with each other
// - action is actually buildable by present worker models
func checkActionRequirements(db database.Querier, proj string, pip string, actionID int64) ([]sdk.Warning, error) {
	warns := []sdk.Warning{}
	var modelReq, hostnameReq int
	var modelName string

	// Load all action requirements
	a, err := action.LoadActionByID(db, actionID)
	if err != nil {
		log.Warning("CheckActionRequirements> Cannot load action %d: %s\n", actionID, err)
		return nil, err
	}

	w, modelReq, modelName, err := checkMultipleWorkerModelWarning(db, proj, pip, a)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	w, hostnameReq, err = checkMultipleHostnameWarning(db, proj, pip, a)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	// Load registered worker model
	wms, err := worker.LoadWorkerModels(db)
	if err != nil {
		log.Warning("CheckActionRequirements> Cannot LoadWorkerModels")
		return nil, err
	}

	w, err = checkNoWorkerModelMatchRequirement(db, proj, pip, a, wms, modelReq, hostnameReq)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	// if we get 1 Model requirement -> check binary requirement with model capabilities
	if modelReq == 1 {
		w, err = checkIncompatibleBinaryWithModelRequirement(db, proj, pip, a, wms, modelName)
		if err != nil {
			return nil, err
		}
		warns = append(warns, w...)
	}

	return warns, nil
}

func checkMultipleWorkerModelWarning(db database.Querier, proj string, pip string, a *sdk.Action) ([]sdk.Warning, int, string, error) {
	var warns []sdk.Warning
	var modelName string
	areqs := a.Requirements

	// If > 1 Model requirement -> WRONG
	var modelReq int
	for _, r := range areqs {
		if r.Type == sdk.ModelRequirement {
			modelReq++
			modelName = r.Value
		}
		if modelReq > 1 {
			modelName = r.Value
			w := sdk.Warning{
				Action: sdk.Action{
					ID: a.ID,
				},
				ID: MultipleWorkerModelWarning,
				MessageParam: map[string]string{
					"ActionName":   a.Name,
					"PipelineName": pip,
					"ProjectKey":   proj,
				},
			}
			warns = append(warns, w)
			break
		}
	}

	return warns, modelReq, modelName, nil
}

func checkMultipleHostnameWarning(db database.Querier, proj string, pip string, a *sdk.Action) ([]sdk.Warning, int, error) {
	var warns []sdk.Warning
	areqs := a.Requirements

	// If > 1 hostname requirement -> WRONG
	var hostnameReq int
	for _, r := range areqs {
		if r.Type == sdk.HostnameRequirement {
			hostnameReq++
		}
	}

	if hostnameReq > 1 {
		w := sdk.Warning{
			Action: sdk.Action{
				ID: a.ID,
			},
			ID: MultipleWorkerModelWarning,
			MessageParam: map[string]string{
				"ActionName":   a.Name,
				"PipelineName": pip,
				"ProjectKey":   proj,
			},
		}
		warns = append(warns, w)
	}

	return warns, hostnameReq, nil
}

func checkNoWorkerModelMatchRequirement(db database.Querier, proj string, pip string, a *sdk.Action, wms []sdk.Model, modelReq int, hostnameReq int) ([]sdk.Warning, error) {
	var warns []sdk.Warning
	areqs := a.Requirements

	// Check all binary requirement are present in at least one model
	validModel := false
	for _, wm := range wms {
		ok := true
		for _, ar := range areqs {
			if ar.Type != sdk.BinaryRequirement {
				continue
			}
			found := false
			for _, wr := range wm.Capabilities {
				if wr.Value == ar.Value {
					found = true
					break
				}
			}
			if !found {
				// Model doesn't have this requirement
				ok = false
				break
			}
		}
		if ok {
			validModel = true
		}
	}
	if !validModel && modelReq == 0 && hostnameReq == 0 {
		w := sdk.Warning{
			Action: sdk.Action{
				ID: a.ID,
			},
			ID: NoWorkerModelMatchRequirement,
			MessageParam: map[string]string{
				"ActionName":   a.Name,
				"PipelineName": pip,
				"ProjectKey":   proj,
			},
		}
		warns = append(warns, w)
	}
	return warns, nil
}

func checkIncompatibleBinaryWithModelRequirement(db database.Querier, proj string, pip string, a *sdk.Action, wms []sdk.Model, modelName string) ([]sdk.Warning, error) {
	var warns []sdk.Warning
	var m sdk.Model
	areqs := a.Requirements

	// find worker model
	for _, wm := range wms {
		if wm.Name == modelName {
			m = wm
			break
		}
	}

	if m.Name == "" { // uh not found
		log.Warning("checkIncompatibleBinaryWithModelRequirement> Model '%s' not found\n", modelName)
		return nil, sdk.ErrNoWorkerModel
	}

	// now for each binary requirement in areqs, check it is found in model capas
	for _, b := range areqs {
		if b.Type != sdk.BinaryRequirement {
			continue
		}

		found := false
		for _, c := range m.Capabilities {
			if c.Type != sdk.BinaryRequirement {
				continue
			}
			if c.Value == b.Value {
				found = true
				break
			}
		}
		if !found {
			w := sdk.Warning{
				Action: sdk.Action{
					ID: a.ID,
				},
				ID: IncompatibleBinaryAndModelRequirements,
				MessageParam: map[string]string{
					"ActionName":        a.Name,
					"PipelineName":      pip,
					"ProjectKey":        proj,
					"ModelName":         modelName,
					"BinaryRequirement": b.Value,
				},
			}
			warns = append(warns, w)
		}
	}

	return warns, nil
}
