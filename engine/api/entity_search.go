package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"
)

type EntityFinder struct {
	currentProject        string
	currentVCS            sdk.VCSProject
	currentRepo           sdk.ProjectRepository
	currentRef            string
	currentSha            string
	vcsServerCache        map[string]sdk.VCSProject
	repoCache             map[string]sdk.ProjectRepository
	repoDefaultRefCache   map[string]string
	actionsCache          map[string]sdk.EntityWithObject
	localActionsCache     map[string]sdk.EntityWithObject
	localWorkerModelCache map[string]sdk.EntityWithObject
	workerModelCache      map[string]sdk.EntityWithObject
	localTemplatesCache   map[string]sdk.EntityWithObject
	templatesCache        map[string]sdk.EntityWithObject
	localWorkflowCache    map[string]sdk.V2Workflow
	workflowCache         map[string]sdk.V2Workflow
	plugins               map[string]sdk.GRPCPlugin
	libraryProject        string
	initiator             sdk.V2Initiator
}

func NewEntityFinder(ctx context.Context, db *gorp.DbMap, pkey, currentRef, currentSha string, repo sdk.ProjectRepository, vcsServer sdk.VCSProject, i sdk.V2Initiator, libraryProjectKey string) (*EntityFinder, error) {
	log.Debug(context.Background(), "NewEntityFinder - initiator: %+v", i)
	ef := &EntityFinder{
		currentProject:        pkey,
		currentVCS:            vcsServer,
		currentRepo:           repo,
		currentRef:            currentRef,
		currentSha:            currentSha,
		actionsCache:          make(map[string]sdk.EntityWithObject),
		localActionsCache:     make(map[string]sdk.EntityWithObject),
		workerModelCache:      make(map[string]sdk.EntityWithObject),
		localWorkerModelCache: make(map[string]sdk.EntityWithObject),
		templatesCache:        make(map[string]sdk.EntityWithObject),
		localTemplatesCache:   make(map[string]sdk.EntityWithObject),
		repoCache:             make(map[string]sdk.ProjectRepository),
		localWorkflowCache:    make(map[string]sdk.V2Workflow),
		workflowCache:         make(map[string]sdk.V2Workflow),
		vcsServerCache:        make(map[string]sdk.VCSProject),
		repoDefaultRefCache:   make(map[string]string),
		plugins:               make(map[string]sdk.GRPCPlugin),
		libraryProject:        libraryProjectKey,
		initiator:             i,
	}

	plugins, err := plugin.LoadAllByType(ctx, db, sdk.GRPCPluginAction)
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		ef.plugins[p.Name] = p
	}
	return ef, nil
}

func (ef *EntityFinder) unsafeSearchEntityFromLibrary(ctx context.Context, db gorp.SqlExecutor, store cache.Store, name string, entityType string) (*sdk.EntityFullName, error) {
	if ef.libraryProject == "" {
		return nil, nil
	}
	var cacheKey = cache.Key("api", "workflowV2", "entityFinder", "library", entityType, name)
	var e *sdk.EntityFullName
	found, err := store.Get(cacheKey, e)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
	}

	if found {
		return e, nil
	}

	log.Debug(ctx, "unsafeSearchEntityFromLibrary> searching for %q  on project %q", name, ef.libraryProject)

	entitiesFullPath, err := entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, db, entityType, []string{ef.libraryProject})
	if err != nil {
		err := sdk.WrapError(err, "invalid workflow: unable to load library entities")
		return nil, err
	}

	for _, entityFullPath := range entitiesFullPath {
		if entityFullPath.Name == name {
			_ = store.Set(cacheKey, entityFullPath)
			return &entityFullPath, nil
		}
	}

	return nil, nil
}

func (ef *EntityFinder) searchEntity(ctx context.Context, db gorp.SqlExecutor, store cache.Store, name string, entityType string) (*sdk.EntityWithObject, string, error) {
	ctx, end := telemetry.Span(ctx, "EntityFinder.searchEntity", trace.StringAttribute("entity-type", entityType), trace.StringAttribute("entity-name", name))
	defer end()

	var ref, branchOrTag, entityName, repoName, vcsName, projKey string

	if name == "" {
		return nil, fmt.Sprintf("unable to find entity of type %s with an empty name", entityType), nil
	}

	// Get branch if present
	splitBranch := strings.Split(name, "@")
	if len(splitBranch) == 2 {
		branchOrTag = splitBranch[1]
	}
	entityFullPath := splitBranch[0]

	entityPathSplit := strings.Split(entityFullPath, "/")
	embeddedEntity := false
	switch len(entityPathSplit) {
	case 1:
		entityName = entityFullPath
		embeddedEntity = true
	case 2:
		if entityPathSplit[0] == "library" {
			entity, err := ef.unsafeSearchEntityFromLibrary(ctx, db, store, entityPathSplit[1], entityType)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
			if entity == nil {
				return nil, fmt.Sprintf("invalid workflow: unable to find %s", entityFullPath), nil
			}
			projKey = entity.ProjectKey
			vcsName = entity.VCSName
			repoName = entity.RepoName
			entityName = entity.Name
			log.Debug(ctx, "searchEntity> matches %q to %s/%s/%s/%s", name, projKey, vcsName, repoName, entityName)
		} else {
			return nil, fmt.Sprintf("invalid workflow: unable to get repository from %s", entityFullPath), nil
		}
	case 3:
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[0], entityPathSplit[1])
		entityName = entityPathSplit[2]
	case 4:
		vcsName = entityPathSplit[0]
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[1], entityPathSplit[2])
		entityName = entityPathSplit[3]
	case 5:
		projKey = entityPathSplit[0]
		vcsName = entityPathSplit[1]
		repoName = fmt.Sprintf("%s/%s", entityPathSplit[2], entityPathSplit[3])
		entityName = entityPathSplit[4]
	default:
		return nil, fmt.Sprintf("unable to parse the %s: %s", entityType, name), nil
	}

	var entityVCS sdk.VCSProject
	var entityRepo sdk.ProjectRepository

	// If no project key in path, get it from workflow run
	if projKey == "" || projKey == ef.currentProject {
		projKey = ef.currentProject
	}

	if !ef.initiator.IsAdminWithMFA {
		can, err := ef.checkEntityReadPermission(ctx, db, projKey)
		if err != nil {
			return nil, "", err
		}
		if !can {
			return nil, fmt.Sprintf("user %s do not have the permission to access %s", ef.initiator.Username(), name), nil
		}
	}

	// If no vcs in path, get it from workflow run
	if vcsName == "" || (vcsName == ef.currentVCS.Name && projKey == ef.currentProject) {
		vcsName = ef.currentVCS.Name
		entityVCS = ef.currentVCS
	} else {
		vcsFromCache, has := ef.vcsServerCache[projKey+"/"+vcsName]
		if has {
			entityVCS = vcsFromCache
		} else {
			vcsDB, err := vcs.LoadVCSByProject(ctx, db, projKey, vcsName)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, fmt.Sprintf("vcs %s not found on project %s for entity path %s", vcsName, projKey, name), nil
				}
				return nil, "", err
			}
			entityVCS = *vcsDB
			ef.vcsServerCache[projKey+"/"+vcsName] = *vcsDB
		}
	}
	// If no repo in path, get it from workflow run
	if repoName == "" || (vcsName == ef.currentVCS.Name && repoName == ef.currentRepo.Name && projKey == ef.currentProject) {
		repoName = ef.currentRepo.Name
		entityRepo = ef.currentRepo
	} else {
		entityFromCache, has := ef.repoCache[projKey+"/"+vcsName+"/"+repoName]
		if has {
			entityRepo = entityFromCache
		} else {
			repoDB, err := repository.LoadRepositoryByName(ctx, db, entityVCS.ID, repoName)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, fmt.Sprintf("repository %s not found on vcs %s into project %s", repoName, vcsName, projKey), nil
				}
				return nil, "", err
			}
			entityRepo = *repoDB
			ef.repoCache[projKey+"/"+vcsName+"/"+repoName] = *repoDB
		}
	}

	if branchOrTag == "" {
		if embeddedEntity || (projKey == ef.currentProject && entityVCS.ID == ef.currentVCS.ID && entityRepo.ID == ef.currentRepo.ID) {
			// Get current git.branch parameters
			ref = ef.currentRef
		} else {
			defaultCache, has := ef.repoDefaultRefCache[projKey+"/"+entityVCS.Name+"/"+entityRepo.Name]
			if has {
				ref = defaultCache
			} else {
				// Get default branch
				client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, projKey, entityVCS.Name)
				if err != nil {
					return nil, "", err
				}
				b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					return nil, "", err
				}
				ref = b.ID
				ef.repoDefaultRefCache[projKey+"/"+entityVCS.Name+"/"+entityRepo.Name] = ref
			}
		}
	} else if strings.HasPrefix(branchOrTag, sdk.GitRefBranchPrefix) || strings.HasPrefix(branchOrTag, sdk.GitRefTagPrefix) {
		ref = branchOrTag
	} else {
		// Need to known if branchOrTag is a tag or a branch
		client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, projKey, entityVCS.Name)
		if err != nil {
			return nil, "", err
		}
		b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{BranchName: branchOrTag})
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, "", err
		}

		if b == nil {
			// try to get tag
			t, err := client.Tag(ctx, entityRepo.Name, branchOrTag)
			if err != nil {
				return nil, "", err
			}
			ref = sdk.GitRefTagPrefix + t.Tag
		} else {
			ref = b.ID
		}
	}

	completePath := fmt.Sprintf("%s/%s/%s/%s", projKey, vcsName, repoName, entityName)
	if ref != "" {
		completePath += "@" + ref
	}

	switch entityType {
	case sdk.EntityTypeAction:
		if act, has := ef.actionsCache[completePath]; has {
			return &act, "", nil
		}
	case sdk.EntityTypeWorkerModel:
		if wm, has := ef.workerModelCache[completePath]; has {
			return &wm, "", nil
		}
	case sdk.EntityTypeWorkflowTemplate:
		if wt, has := ef.templatesCache[completePath]; has {
			return &wt, "", nil
		}
	}

	var entityDB *sdk.Entity
	var err error
	if projKey != ef.currentProject || entityVCS.Name != ef.currentVCS.Name || entityRepo.Name != ef.currentRepo.Name || ref != ef.currentRef {
		entityDB, err = entity.LoadHeadEntityByRefTypeName(ctx, db, entityRepo.ID, ref, entityType, entityName)
	} else {
		entityDB, err = entity.LoadByRefTypeNameCommit(ctx, db, entityRepo.ID, ef.currentRef, entityType, entityName, ef.currentSha)
	}
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, fmt.Sprintf("unable to find workflow dependency: %s", name), nil
		}
		return nil, "", err
	}

	var eo *sdk.EntityWithObject
	switch entityType {
	case sdk.EntityTypeAction:
		var act sdk.V2Action
		if err := yaml.Unmarshal([]byte(entityDB.Data), &act); err != nil {
			return nil, "", err
		}
		eo = &sdk.EntityWithObject{Entity: *entityDB, Action: act, CompleteName: completePath}
		ef.actionsCache[completePath] = *eo
	case sdk.EntityTypeWorkerModel:
		var wm sdk.V2WorkerModel
		if err := yaml.Unmarshal([]byte(entityDB.Data), &wm); err != nil {
			return nil, "", err
		}
		eo = &sdk.EntityWithObject{Entity: *entityDB, Model: wm, CompleteName: completePath}
		if err := eo.Interpolate(ctx); err != nil {
			return nil, "", err
		}
		ef.workerModelCache[completePath] = *eo
	case sdk.EntityTypeWorkflowTemplate:
		var wt sdk.V2WorkflowTemplate
		if err := yaml.Unmarshal([]byte(entityDB.Data), &wt); err != nil {
			return nil, "", err
		}
		eo = &sdk.EntityWithObject{
			Entity:       *entityDB,
			Template:     wt,
			CompleteName: completePath,
		}
		ef.templatesCache[completePath] = *eo
	case sdk.EntityTypeWorkflow:
		var w sdk.V2Workflow
		if err := yaml.Unmarshal([]byte(entityDB.Data), &w); err != nil {
			return nil, "", err
		}
		ef.workflowCache[completePath] = w
		eo = &sdk.EntityWithObject{
			Entity:       *entityDB,
			Workflow:     w,
			CompleteName: completePath,
		}

	default:
		return nil, "", sdk.NewErrorFrom(sdk.ErrNotImplemented, "entity %s not implemented", entityType)
	}
	return eo, "", nil
}

func (ef *EntityFinder) searchAction(ctx context.Context, db gorp.SqlExecutor, store cache.Store, name string) (*sdk.EntityWithObject, string, error) {
	// Local def
	if strings.HasPrefix(name, ".cds/actions/") {
		// Find action from path
		localAct, has := ef.localActionsCache[name]
		if !has {
			actionEntity, err := entity.LoadEntityByPathAndRefAndCommit(ctx, db, ef.currentRepo.ID, name, ef.currentRef, ef.currentSha)
			if err != nil {
				return nil, fmt.Sprintf("Unable to find action %s", name), nil
			}
			localAct.Entity = *actionEntity
			if err := yaml.Unmarshal([]byte(actionEntity.Data), &localAct.Action); err != nil {
				return nil, "", err
			}
			if !ef.initiator.IsAdminWithMFA {
				can, err := ef.checkEntityReadPermission(ctx, db, actionEntity.ProjectKey)
				if err != nil {
					return nil, "", err
				}
				if !can {
					return nil, fmt.Sprintf("user %s do not have the permission to access %s", ef.initiator.Username(), name), nil
				}
			}
			localAct.CompleteName = fmt.Sprintf("%s/%s/%s/%s@%s", localAct.ProjectKey, ef.currentVCS.Name, ef.currentRepo.Name, localAct.Name, ef.currentRef)
			ef.localActionsCache[name] = localAct
		}
		return &localAct, "", nil
	}

	actionName := strings.TrimPrefix(name, "actions/")
	actionSplit := strings.Split(actionName, "/")

	// If plugins
	if strings.HasPrefix(name, "actions/") && len(actionSplit) == 1 {
		// Check plugins
		if _, has := ef.plugins[actionSplit[0]]; !has {
			return nil, fmt.Sprintf("Action %s doesn't exist", actionSplit[0]), nil
		}
		return nil, "", nil
	}

	// Others
	entityWithObj, msg, err := ef.searchEntity(ctx, db, store, actionName, sdk.EntityTypeAction)
	if msg != "" || err != nil {
		return nil, msg, err
	}
	return entityWithObj, msg, err
}

func (ef *EntityFinder) searchWorkerModel(ctx context.Context, db gorp.SqlExecutor, store cache.Store, name string) (*sdk.EntityWithObject, string, error) {
	// Local def
	if strings.HasPrefix(name, ".cds/worker-models/") {
		// Find worker model from path
		localWM, has := ef.localWorkerModelCache[name]
		if !has {
			wmEntity, err := entity.LoadEntityByPathAndRefAndCommit(ctx, db, ef.currentRepo.ID, name, ef.currentRef, ef.currentSha)
			if err != nil {
				return nil, fmt.Sprintf("Unable to find worker model %s in repository %s", name, ef.currentRepo.Name), nil
			}
			var wm sdk.V2WorkerModel
			if err := yaml.Unmarshal([]byte(wmEntity.Data), &wm); err != nil {
				return nil, "", err
			}

			if !ef.initiator.IsAdminWithMFA {
				can, err := ef.checkEntityReadPermission(ctx, db, wmEntity.ProjectKey)
				if err != nil {
					return nil, "", err
				}
				if !can {
					return nil, fmt.Sprintf("user %s do not have the permission to access %s", ef.initiator.Username(), name), nil
				}
			}

			completeName := fmt.Sprintf("%s/%s/%s/%s@%s", ef.currentProject, ef.currentVCS.Name, ef.currentRepo.Name, wm.Name, ef.currentRef)
			localWM = sdk.EntityWithObject{Entity: *wmEntity, Model: wm, CompleteName: completeName}
			ef.localWorkerModelCache[name] = localWM
		}

		return &localWM, "", nil
	}

	entityWithObj, msg, err := ef.searchEntity(ctx, db, store, name, sdk.EntityTypeWorkerModel)
	if err != nil {
		return nil, "", err
	}
	if msg != "" {
		return nil, msg, nil
	}
	return entityWithObj, "", nil
}

func (ef *EntityFinder) searchWorkflowTemplate(ctx context.Context, db gorp.SqlExecutor, store cache.Store, name string) (*sdk.EntityWithObject, string, error) {
	if strings.HasPrefix(name, ".cds/workflow-templates/") {
		// Find tempalte from path
		localEntity, has := ef.localTemplatesCache[name]
		if !has {
			wtEntity, err := entity.LoadEntityByPathAndRefAndCommit(ctx, db, ef.currentRepo.ID, name, ef.currentRef, ef.currentSha)
			if err != nil {
				msg := fmt.Sprintf("Unable to find workflow template %s %s %s %s", ef.currentRepo.ID, name, ef.currentRef, ef.currentSha)
				return nil, msg, nil
			}
			if err := yaml.Unmarshal([]byte(wtEntity.Data), &localEntity.Template); err != nil {
				return nil, "", sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to read workflow template %s: %v", name, err)
			}
			if !ef.initiator.IsAdminWithMFA {
				can, err := ef.checkEntityReadPermission(ctx, db, wtEntity.ProjectKey)
				if err != nil {
					return nil, "", err
				}
				if !can {
					return nil, fmt.Sprintf("user %s do not have the permission to access %s", ef.initiator.Username(), name), nil
				}
			}
			localEntity.Entity = *wtEntity
			localEntity.CompleteName = fmt.Sprintf("%s/%s/%s/%s@%s", ef.currentProject, ef.currentVCS.Name, ef.currentRepo.Name, localEntity.Template.Name, ef.currentRef)
			ef.localTemplatesCache[name] = localEntity
		}
		return &localEntity, "", nil
	}
	entityWithObj, msg, err := ef.searchEntity(ctx, db, store, name, sdk.EntityTypeWorkflowTemplate)
	if err != nil {
		return nil, "", err
	}
	if msg != "" {
		return nil, msg, nil
	}
	return entityWithObj, "", nil

}

func (ef *EntityFinder) checkEntityReadPermission(ctx context.Context, db gorp.SqlExecutor, projKey string) (bool, error) {
	// Verify project read permission
	if ef.initiator.IsUser() {
		can, err := rbac.HasRoleOnProjectAndUserID(ctx, db, sdk.ProjectRoleRead, ef.initiator.UserID, projKey)
		if err != nil {
			return false, err
		}
		return can, nil
	}
	can, err := rbac.HasRoleOnProjectAndVCSUser(ctx, db, sdk.ProjectRoleRead, sdk.RBACVCSUser{VCSServer: ef.initiator.VCS, VCSUsername: ef.initiator.VCSUsername}, projKey)
	if err != nil {
		return false, err
	}
	return can, nil

}
