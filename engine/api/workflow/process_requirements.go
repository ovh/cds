package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/repository"
	vcs2 "github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// processNodeJobRunRequirements returns requirements list interpolated, and true or false if at least
// one requirement is of type "Service"
func processNodeJobRunRequirements(ctx context.Context, db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, execsGroupIDs []int64, integrationPlugins []sdk.GRPCPlugin, integrationsConfigs []sdk.IntegrationConfig) (sdk.RequirementList, bool, string, *sdk.MultiError) {
	var requirements sdk.RequirementList
	var errm sdk.MultiError
	var containsService bool
	var model string
	var modelV2 string
	var modelType string
	var wm *sdk.Model
	var tmp = sdk.ParametersToMap(run.BuildParameters)

	if defaultOS != "" && defaultArch != "" {
		var modelFound, osArchFound bool
		for _, req := range j.Action.Requirements {
			if req.Type == sdk.ModelRequirement || req.Type == sdk.ModelV2Requirement {
				modelFound = true
			}
			if req.Type == sdk.OSArchRequirement {
				osArchFound = true
			}
		}
		if !modelFound && !osArchFound {
			j.Action.Requirements = append(j.Action.Requirements, sdk.Requirement{
				Name:  defaultOS + "/" + defaultArch,
				Type:  sdk.OSArchRequirement,
				Value: defaultOS + "/" + defaultArch,
			})
		}
	}

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
		if v.Type == sdk.ModelRequirement || v.Type == sdk.ModelV2Requirement {
			// It is forbidden to have more than one model requirement.
			if j.Action.Enabled && model != "" {
				errm.Append(sdk.ErrInvalidJobRequirementDuplicateModel)
				break
			}
			if v.Type == sdk.ModelRequirement {
				model = value
			}
			if v.Type == sdk.ModelV2Requirement {
				modelV2 = value
			}

		}
		sdk.AddRequirement(&requirements, v.ID, name, v.Type, value)
	}

	if model != "" {
		var err error
		wm, err = processNodeJobRunRequirementsGetModel(ctx, db, model, execsGroupIDs)
		if err != nil {
			log.Error(ctx, "getNodeJobRunRequirements> error while getting worker model %s: %v", model, err)
			errm.Append(err)
		}
	}
	if modelV2 != "" {
		workerModelV2, err := processNodeJobRunRequirementsGetModelV2(ctx, db, modelV2)
		if err != nil {
			log.Error(ctx, "getNodeJobRunRequirements> error while getting worker model v2 %s: %v", modelV2, err)
			errm.Append(sdk.NewErrorFrom(sdk.ErrInvalidJobRequirement, "unable to get worker model %s", modelV2))
		}
		if workerModelV2 != nil {
			modelType = workerModelV2.Type
		}

	}

	if wm != nil {
		modelType = wm.Type

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

	// Add plugin requirement if needed based on the os/arch of the job
	if len(integrationPlugins) > 0 {
		var os, arch string

		// Compute os/arch values from Model or OSArch requirement
		if wm != nil {
			if !wm.NeedRegistration && !wm.CheckRegistration {
				os = *wm.RegisteredOS
				arch = *wm.RegisteredArch
			}
		} else {
			for i := range requirements {
				if requirements[i].Type == sdk.OSArchRequirement {
					osarch := strings.Split(requirements[i].Value, "/")
					if len(osarch) != 2 {
						errm.Append(fmt.Errorf("invalid requirement %s", requirements[i].Value))
					} else {
						os = strings.ToLower(osarch[0])
						arch = strings.ToLower(osarch[1])
					}
					break
				}
			}
		}

		// If os/arch values were found adding requirements from plugin binary
		if os != "" && arch != "" {
			for _, p := range integrationPlugins {
				for _, b := range p.Binaries {
					if strings.ToLower(b.OS) == os && strings.ToLower(b.Arch) == arch {
						for i := range b.Requirements {
							sdk.AddRequirement(&requirements, b.Requirements[i].ID, b.Requirements[i].Name, b.Requirements[i].Type, b.Requirements[i].Value)
						}
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
		return requirements, containsService, modelType, nil
	}
	return requirements, containsService, modelType, &errm
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

func processNodeJobRunRequirementsGetModelV2(ctx context.Context, db gorp.SqlExecutor, requirementValue string) (*sdk.V2WorkerModel, error) {
	modelPath := strings.Split(requirementValue, "/")
	if len(modelPath) < 4 {
		return nil, sdk.WrapError(sdk.ErrInvalidData, "wrong model value %v", modelPath)
	}
	projKey := modelPath[0]
	vcsName := modelPath[1]
	modelName := modelPath[len(modelPath)-1]
	repoName := strings.Join(modelPath[2:len(modelPath)-1], "/")

	vcs, err := vcs2.LoadVCSByName(ctx, db, projKey, vcsName)
	if err != nil {
		return nil, err
	}
	repo, err := repository.LoadRepositoryByName(ctx, db, vcs.ID, repoName)
	if err != nil {
		return nil, err
	}
	workerModelEntity, err := entity.LoadByBranchTypeName(ctx, db, repo.ID, "master", sdk.EntityTypeWorkerModel, modelName)
	if err != nil {
		return nil, err
	}
	var model sdk.V2WorkerModel
	if err := yaml.Unmarshal([]byte(workerModelEntity.Data), &model); err != nil {
		return nil, err
	}
	return &model, nil
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
