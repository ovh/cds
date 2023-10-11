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
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	vcs2 "github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
)

// processNodeJobRunRequirements returns requirements list interpolated, and true or false if at least
// one requirement is of type "Service"
func processNodeJobRunRequirements(ctx context.Context, store cache.Store, db gorpmapper.SqlExecutorWithTx, projectKey string, wr sdk.WorkflowRun, j sdk.Job, run *sdk.WorkflowNodeRun, execsGroupIDs []int64, integrationPlugins []sdk.GRPCPlugin, integrationsConfigs []sdk.IntegrationConfig, jobParams []sdk.Parameter) (sdk.RequirementList, bool, string, *sdk.MultiError) {
	var requirements sdk.RequirementList
	var errm sdk.MultiError
	var containsService bool
	var model string
	var modelType string
	var wm *sdk.Model
	var tmp = sdk.ParametersToMap(run.BuildParameters)

	if defaultOS != "" && defaultArch != "" {
		var modelFound, osArchFound bool
		for _, req := range j.Action.Requirements {
			if req.Type == sdk.ModelRequirement {
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
		if v.Type == sdk.ModelRequirement {
			// It is forbidden to have more than one model requirement.
			if j.Action.Enabled && model != "" {
				errm.Append(sdk.ErrInvalidJobRequirementDuplicateModel)
				break
			}
			if v.Type == sdk.ModelRequirement {
				model = value

				var err error
				wm, err = processNodeJobRunRequirementsGetModel(ctx, db, model, execsGroupIDs)
				if err != nil {
					if sdk.ErrorIs(err, sdk.ErrNotFound) {
						workerModelV2, workerModelFullPath, err := processNodeJobRunRequirementsGetModelV2(ctx, store, db, projectKey, wr, model, jobParams)
						if err != nil {
							log.Error(ctx, "getNodeJobRunRequirements> error while getting worker model %s: %v", model, err)
							errm.Append(sdk.NewErrorFrom(sdk.ErrInvalidJobRequirement, "unable to get worker model %s", model))
						}
						if workerModelV2 != nil {
							modelType = workerModelV2.Type
							value = workerModelFullPath
						}
					} else {
						log.Error(ctx, "getNodeJobRunRequirements> error while getting worker model %s: %v", model, err)
						errm.Append(err)
					}
				}
			}
		}
		sdk.AddRequirement(&requirements, v.ID, name, v.Type, value)
	}
	if wm != nil {
		if wm.Disabled {
			errm.Append(sdk.NewErrorFrom(sdk.ErrInvalidData, "worker model %s is disabled. Please update your requirement", wm.Name))
		}
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

func processNodeJobRunRequirementsGetModelV2(ctx context.Context, store cache.Store, db gorpmapper.SqlExecutorWithTx, projectKey string, wr sdk.WorkflowRun, requirementValue string, jobParams []sdk.Parameter) (*sdk.V2WorkerModel, string, error) {
	var workerProjKey, vcsName, repoName, workerModelName, branch string
	var app sdk.Application

	// complete format <ProjKey>/<VCSServer>/<RepoSLUG/RepoName>/<WorkerModelName@Branch

	// Get branch if present
	splitBranch := strings.Split(requirementValue, "@")
	if len(splitBranch) == 2 {
		branch = splitBranch[1]
	}
	modelFullPath := splitBranch[0]

	// Search application on root node
	nodeContext := wr.Workflow.WorkflowData.Node.Context
	if nodeContext.ApplicationID != 0 {
		app = wr.Workflow.Applications[nodeContext.ApplicationID]
	}

	modelPathSplit := strings.Split(modelFullPath, "/")
	embeddedModel := false
	switch len(modelPathSplit) {
	case 1:
		workerModelName = modelFullPath
		embeddedModel = true
	case 2:
		return nil, "", sdk.WrapError(sdk.ErrInvalidData, "unable to find repository for this worker model")
	case 3:
		repoName = fmt.Sprintf("%s/%s", modelPathSplit[0], modelPathSplit[1])
		workerModelName = modelPathSplit[2]
	case 4:
		vcsName = modelPathSplit[0]
		repoName = fmt.Sprintf("%s/%s", modelPathSplit[1], modelPathSplit[2])
		workerModelName = modelPathSplit[3]
	case 5:
		workerProjKey = modelPathSplit[0]
		vcsName = modelPathSplit[1]
		repoName = fmt.Sprintf("%s/%s", modelPathSplit[2], modelPathSplit[3])
		workerModelName = modelPathSplit[4]
	default:
		return nil, "", sdk.WrapError(sdk.ErrInvalidData, "unable to handle the worker model requirement")
	}

	if workerProjKey == "" {
		workerProjKey = projectKey
	}
	if vcsName == "" {
		vcsName = app.VCSServer
	}
	if repoName == "" {
		repoName = app.RepositoryFullname
	}

	if vcsName == "" || repoName == "" {
		return nil, "", sdk.WrapError(sdk.ErrInvalidData, "unable to retrieve worker model data because the workflow root pipeline does not contain any vcs configuration")
	}

	vcs, err := vcs2.LoadVCSByProject(ctx, db, workerProjKey, vcsName)
	if err != nil {
		return nil, "", err
	}
	repo, err := repository.LoadRepositoryByName(ctx, db, vcs.ID, repoName)
	if err != nil {
		return nil, "", err
	}
	if branch == "" {
		if embeddedModel {
			// Get current git.branch parameters
			for _, p := range jobParams {
				if p.Name == "git.branch" {
					branch = p.Value
				}
			}
			if branch == "" {
				return nil, "", sdk.NewErrorFrom(sdk.ErrNotFound, "worker model %s not found on the current branch", workerModelName)
			}
		} else {
			// Get default branch
			client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, workerProjKey, vcs.Name)
			if err != nil {
				return nil, "", err
			}
			b, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
			if err != nil {
				return nil, "", err
			}
			branch = b.DisplayID
		}
	}
	workerModelEntity, err := entity.LoadByBranchTypeName(ctx, db, repo.ID, branch, sdk.EntityTypeWorkerModel, workerModelName)
	if err != nil {
		return nil, "", err
	}
	var model sdk.V2WorkerModel
	if err := yaml.Unmarshal([]byte(workerModelEntity.Data), &model); err != nil {
		return nil, "", err
	}

	completePath := fmt.Sprintf("%s/%s/%s/%s", workerProjKey, vcsName, repoName, workerModelName)
	if branch != "" {
		completePath += "@" + branch
	}
	return &model, completePath, nil
}

func processNodeJobRunRequirementsGetModel(ctx context.Context, db gorp.SqlExecutor, model string, execsGroupIDs []int64) (*sdk.Model, error) {
	if model == "" {
		return nil, nil
	}

	var wm *sdk.Model

	modelName := strings.Split(model, " ")[0]
	modelPath := strings.Split(modelName, "/")
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
