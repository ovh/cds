package sanity

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CheckActionRequirements checks:
// - requirements are compabtible with each other
// - action is actually buildable by present worker models
func checkActionRequirements(a *sdk.Action, proj string, pip string, wms []sdk.Model) ([]sdk.Warning, error) {
	warns := []sdk.Warning{}
	var modelReq, hostnameReq int
	var modelName string

	w, modelReq, modelName, err := checkMultipleWorkerModelWarning(proj, pip, a)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	w, hostnameReq, err = checkMultipleHostnameWarning(proj, pip, a)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	w, err = checkNoWorkerModelMatchRequirement(proj, pip, a, wms, modelReq, hostnameReq)
	if err != nil {
		return nil, err
	}
	warns = append(warns, w...)

	// if we get 1 Model requirement -> check binary requirement with model capabilities
	if modelReq == 1 {
		w, err = checkIncompatibleBinaryWithModelRequirement(proj, pip, a, wms, modelName)
		if err != nil {
			return nil, err
		}
		warns = append(warns, w...)

		w, err = checkIncompatibleServiceWithModelRequirement(proj, pip, a, wms, modelName)
		if err != nil {
			return nil, err
		}
		warns = append(warns, w...)

		w, err = checkIncompatibleMemoryWithModelRequirement(proj, pip, a, wms, modelName)
		if err != nil {
			return nil, err
		}
		warns = append(warns, w...)
	}

	return warns, nil
}

func checkMultipleWorkerModelWarning(proj string, pip string, a *sdk.Action) ([]sdk.Warning, int, string, error) {
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

func checkMultipleHostnameWarning(proj string, pip string, a *sdk.Action) ([]sdk.Warning, int, error) {
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
			ID: MultipleHostnameRequirement,
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

func checkNoWorkerModelMatchRequirement(proj string, pip string, a *sdk.Action, wms []sdk.Model, modelReq int, hostnameReq int) ([]sdk.Warning, error) {
	var warns []sdk.Warning
	areqs := a.Requirements

	// Check all binary requirement are present in at least one model
	validModel := false
	for _, wm := range wms {
		ok := true
		for _, ar := range areqs {
			//Service requirements and Memory requirements are only compliant with docker worker models
			if (ar.Type == sdk.ServiceRequirement || ar.Type == sdk.MemoryRequirement) && wm.Type != sdk.Docker {
				// Model doesn't have this requirement
				ok = false
				break
			}

			// We are only checkins binary requirement matching with binary capabilities
			// so let's skip this other types of requirements
			if ar.Type != sdk.BinaryRequirement {
				continue
			}

			found := false
			for _, wr := range wm.RegisteredCapabilities {
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

func checkIncompatibleBinaryWithModelRequirement(proj string, pip string, a *sdk.Action, wms []sdk.Model, modelName string) ([]sdk.Warning, error) {
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
		for _, c := range m.RegisteredCapabilities {
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

func checkIncompatibleServiceWithModelRequirement(proj string, pip string, a *sdk.Action, wms []sdk.Model, modelName string) ([]sdk.Warning, error) {
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

	if m.Name == "" {
		log.Warning("checkIncompatibleServiceWithModelRequirement> Model '%s' not found\n", modelName)
		return nil, sdk.ErrNoWorkerModel
	}

	var service *sdk.Requirement

	// now for each binary requirement in areqs, check it is found in model capas
	for i, b := range areqs {
		if b.Type == sdk.ServiceRequirement {
			service = &areqs[i]
			break
		}
	}

	if service != nil {
		if m.Type == sdk.Docker {
			return nil, nil
		}
		w := sdk.Warning{
			Action: sdk.Action{
				ID: a.ID,
			},
			ID: IncompatibleServiceAndModelRequirements,
			MessageParam: map[string]string{
				"ActionName":         a.Name,
				"PipelineName":       pip,
				"ProjectKey":         proj,
				"ModelName":          modelName,
				"ServiceRequirement": service.Value,
			},
		}
		warns = append(warns, w)
	}

	return warns, nil
}

func checkIncompatibleMemoryWithModelRequirement(proj string, pip string, a *sdk.Action, wms []sdk.Model, modelName string) ([]sdk.Warning, error) {
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

	if m.Name == "" {
		log.Warning("checkIncompatibleMemoryWithModelRequirement> Model '%s' not found\n", modelName)
		return nil, sdk.ErrNoWorkerModel
	}

	var req *sdk.Requirement
	// now for each binary requirement in areqs, check it is found in model capas
	for i, b := range areqs {
		if b.Type == sdk.MemoryRequirement {
			req = &areqs[i]
			break
		}
	}

	if req != nil {
		if m.Type == sdk.Docker {
			return nil, nil
		}
		w := sdk.Warning{
			Action: sdk.Action{
				ID: a.ID,
			},
			ID: IncompatibleMemoryAndModelRequirements,
			MessageParam: map[string]string{
				"ActionName":   a.Name,
				"PipelineName": pip,
				"ProjectKey":   proj,
				"ModelName":    modelName,
			},
		}
		warns = append(warns, w)
	}

	return warns, nil
}
