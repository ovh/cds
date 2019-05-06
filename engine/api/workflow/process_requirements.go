package workflow

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

// processNodeJobRunRequirements returns requirements list interpolated, and true or false if at least
// one requirement is of type "Service"
func processNodeJobRunRequirements(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, execsGroupIDs []int64, integrationPluginBinaries []sdk.GRPCPluginBinary) (sdk.RequirementList, bool, string, *sdk.MultiError) {
	var requirements sdk.RequirementList
	var errm sdk.MultiError
	var containsService bool
	var model string
	var tmp = sdk.ParametersToMap(run.BuildParameters)

	for i := range integrationPluginBinaries {
		j.Action.Requirements = append(j.Action.Requirements, integrationPluginBinaries[i].Requirements...)
	}

	for _, v := range j.Action.Requirements {
		name, errName := interpolate.Do(v.Name, tmp)
		if errName != nil {
			errm.Append(errName)
			continue
		}
		value, errValue := interpolate.Do(v.Value, tmp)
		if errValue != nil {
			errm.Append(errValue)
			continue
		}

		if v.Type == sdk.ServiceRequirement {
			containsService = true
		}
		if v.Type == sdk.ModelRequirement {
			// It is forbidden to have more than one model requirement.
			if model != "" {
				errm.Append(sdk.ErrInvalidJobRequirementDuplicateModel)
				break
			}
			model = value
		}

		sdk.AddRequirement(&requirements, v.ID, name, v.Type, value)
	}

	var modelType string
	if model != "" {
		// Load the worker model
		wm, err := worker.LoadWorkerModelByName(db, strings.Split(model, " ")[0])
		if err != nil {
			log.Error("getNodeJobRunRequirements> error while getting worker model %s: %v", model, err)
			errm.Append(sdk.ErrNoWorkerModel)
		} else {
			// Check that the worker model is in an exec group
			if !sdk.IsInInt64Array(wm.GroupID, execsGroupIDs) {
				errm.Append(sdk.ErrInvalidJobRequirementWorkerModelPermission)
			}

			// Check that the worker model has the binaries capabilitites
			// only if the worker model doesn't need registration
			if !wm.NeedRegistration && !wm.CheckRegistration {
				for _, req := range requirements {
					if req.Type == sdk.BinaryRequirement {
						var hasCapa bool
						for _, cap := range wm.RegisteredCapabilities {
							if cap.Value == req.Value {
								hasCapa = true
								break
							}
						}
						if !hasCapa {
							errm.Append(sdk.ErrInvalidJobRequirementWorkerModelCapabilitites)
							break
						}
					}
				}
			}

			modelType = wm.Type
		}

	}

	if errm.IsEmpty() {
		return requirements, containsService, modelType, nil
	}
	return requirements, containsService, modelType, &errm
}

func prepareRequirementsToNodeJobRunParameters(reqs sdk.RequirementList) []sdk.Parameter {
	params := []sdk.Parameter{}
	for _, r := range reqs {
		if r.Type == sdk.ServiceRequirement {
			k := fmt.Sprintf("job.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
			values := strings.Split(r.Value, " ")
			if len(values) > 1 {
				sdk.AddParameter(&params, k+".image", sdk.StringParameter, values[0])
				sdk.AddParameter(&params, k+".options", sdk.StringParameter, strings.Join(values[1:], " "))
			}
		}
		k := fmt.Sprintf("job.requirement.%s.%s", strings.ToLower(r.Type), strings.ToLower(r.Name))
		sdk.AddParameter(&params, k, sdk.StringParameter, r.Value)
	}
	return params
}
