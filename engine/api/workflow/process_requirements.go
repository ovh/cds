package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// processNodeJobRunRequirements returns requirements list interpolated, and true or false if at least
// one requirement is of type "Service"
func processNodeJobRunRequirements(ctx context.Context, db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, execsGroupIDs []int64, integrationPlugins []sdk.GRPCPlugin, integrationsConfigs []sdk.IntegrationConfig) (sdk.RequirementList, bool, *sdk.Model, *sdk.MultiError) {
	var requirements sdk.RequirementList
	var errm sdk.MultiError
	var containsService bool
	var model string
	var tmp = sdk.ParametersToMap(run.BuildParameters)

	pluginsRequirements := make([]sdk.Requirement, 0)
	for _, p := range integrationPlugins {
		for _, b := range p.Binaries {
			pluginsRequirements = append(pluginsRequirements, b.Requirements...)
		}
	}

	// then add plugins requirement to the action requirement
	j.Action.Requirements = append(j.Action.Requirements, pluginsRequirements...)

	integrationRequirements := make([]sdk.Requirement, 0)
	for _, c := range integrationsConfigs {
		for k, v := range c {
			if v.Type != sdk.IntegrationConfigTypeRegion {
				continue
			}
			integrationRequirements = append(integrationRequirements, sdk.Requirement{
				Name:  k,
				Type:  sdk.RegionRequirement,
				Value: v.Value,
			})
		}
	}
	j.Action.Requirements = append(j.Action.Requirements, integrationRequirements...)

	j.Action.Requirements = sdk.RequirementListDeduplicate(j.Action.Requirements)

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
			if j.Action.Enabled && model != "" {
				errm.Append(sdk.ErrInvalidJobRequirementDuplicateModel)
				break
			}
			model = value
		}

		sdk.AddRequirement(&requirements, v.ID, name, v.Type, value)
	}

	wm, err := processNodeJobRunRequirementsGetModel(ctx, db, model, execsGroupIDs)
	if err != nil {
		log.Error(ctx, "getNodeJobRunRequirements> error while getting worker model %s: %v", model, err)
		errm.Append(err)
	}
	if wm != nil {
		// Check that the worker model has the binaries capabilitites
		// only if the worker model doesn't need registration
		if !wm.NeedRegistration && !wm.CheckRegistration {
			for _, req := range requirements {
				if req.Type == sdk.BinaryRequirement {
					var hasCapa bool
					for _, capa := range wm.RegisteredCapabilities {
						if capa.Value == req.Value {
							hasCapa = true
							break
						}
					}
					if j.Action.Enabled && !hasCapa {
						errm.Append(sdk.ErrInvalidJobRequirementWorkerModelCapabilitites)
						break
					}
				}
			}
		}
	}

	regionRequirementMap := make(map[string]struct{})
	for _, r := range requirements {
		if r.Type != sdk.RegionRequirement {
			continue
		}
		if _, has := regionRequirementMap[r.Value]; !has {
			regionRequirementMap[r.Value] = struct{}{}
		}
	}
	if len(regionRequirementMap) > 1 {
		errm.Append(sdk.NewErrorFrom(sdk.ErrInvalidJobRequirement, "Cannot have multiple region requirements %v", regionRequirementMap))
	}

	if errm.IsEmpty() {
		return requirements, containsService, wm, nil
	}
	return requirements, containsService, wm, &errm
}

func prepareRequirementsToNodeJobRunParameters(reqs sdk.RequirementList) []sdk.Parameter {
	params := make([]sdk.Parameter, 0)
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

func processNodeJobRunRequirementsGetModel(ctx context.Context, db gorp.SqlExecutor, model string, execsGroupIDs []int64) (*sdk.Model, error) {
	if model == "" {
		return nil, nil
	}

	var wm *sdk.Model

	modelName := strings.Split(model, " ")[0]
	modelPath := strings.SplitN(modelName, "/", 2)
	if len(modelPath) == 2 {
		// if model contains group name (myGroup/myModel), try to find the model for the
		g, err := group.LoadByName(ctx, db, modelPath[0])
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "could not find a worker model that match %s", modelName)
			}
			return nil, err
		}

		if !sdk.IsInInt64Array(g.ID, execsGroupIDs) {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidJobRequirementWorkerModelPermission, "group %s should have execution permission", g.Name)
		}

		wm, err = workermodel.LoadByNameAndGroupID(ctx, db, modelPath[1], g.ID, workermodel.LoadOptions.Default)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "could not find a worker model that match %s", modelName)
			}
			return nil, err
		}
	} else {
		var err error

		// if there is no group info, try to find a shared.infra model for given name
		wm, err = workermodel.LoadByNameAndGroupID(ctx, db, modelName, group.SharedInfraGroup.ID, workermodel.LoadOptions.Default)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}

		// if there is no shared.infra model we will try to find one for exec groups, backward compatibility for existing workflow runs.
		if wm == nil {
			wms, err := workermodel.LoadAllByNameAndGroupIDs(ctx, db, modelName, execsGroupIDs, workermodel.LoadOptions.Default)
			if err != nil {
				return nil, err
			}
			if len(wms) > 1 {
				return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "invalid given model name \"%s\", missing group name in requirement", modelName)
			}
			if len(wms) == 0 {
				return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "can not find a model with name \"%s\" for workflow's exec groups", modelName)
			}
			wm = &wms[0]
		}
	}

	return wm, nil
}
