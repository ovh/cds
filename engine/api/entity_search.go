package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"
)

type EntityFinder struct {
	currentProject        string
	currentUserID         string
	currentUserName       string
	currentVCS            sdk.VCSProject
	currentRepo           sdk.ProjectRepository
	currentRef            string
	currentSha            string
	vcsServerCache        map[string]sdk.VCSProject
	repoCache             map[string]sdk.ProjectRepository
	repoDefaultRefCache   map[string]string
	actionsCache          map[string]sdk.V2Action
	localActionsCache     map[string]sdk.V2Action
	localWorkerModelCache map[string]sdk.V2WorkerModel
	workerModelCache      map[string]sdk.V2WorkerModel
	localTemplatesCache   map[string]sdk.EntityWithObject
	templatesCache        map[string]sdk.EntityWithObject
	plugins               map[string]sdk.GRPCPlugin
}

func NewEntityFinder(pkey, currentRef, currentSha string, repo sdk.ProjectRepository, vcsServer sdk.VCSProject, u sdk.AuthentifiedUser) *EntityFinder {
	return &EntityFinder{
		currentProject:        pkey,
		currentUserID:         u.ID,
		currentUserName:       u.Username,
		currentVCS:            vcsServer,
		currentRepo:           repo,
		currentRef:            currentRef,
		currentSha:            currentSha,
		actionsCache:          make(map[string]sdk.V2Action),
		localActionsCache:     make(map[string]sdk.V2Action),
		workerModelCache:      make(map[string]sdk.V2WorkerModel),
		localWorkerModelCache: make(map[string]sdk.V2WorkerModel),
		templatesCache:        make(map[string]sdk.EntityWithObject),
		localTemplatesCache:   make(map[string]sdk.EntityWithObject),
		repoCache:             make(map[string]sdk.ProjectRepository),
		vcsServerCache:        make(map[string]sdk.VCSProject),
		repoDefaultRefCache:   make(map[string]string),
		plugins:               make(map[string]sdk.GRPCPlugin),
	}
}

func (ef *EntityFinder) searchEntity(ctx context.Context, db *gorp.DbMap, store cache.Store, name string, entityType string) (string, string, error) {
	ctx, end := telemetry.Span(ctx, "EntityFinder.searchEntity", trace.StringAttribute("entity-type", entityType), trace.StringAttribute("entity-name", name))
	defer end()

	var ref, branchOrTag, entityName, repoName, vcsName, projKey string

	if name == "" {
		return "", fmt.Sprintf("unable to find entity of type %s with an empty name", entityType), nil
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
		return "", fmt.Sprintf("invalid workflow: unable to get repository from %s", entityFullPath), nil
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
		return "", fmt.Sprintf("unable to parse the %s: %s", entityType, name), nil
	}

	var entityVCS sdk.VCSProject
	var entityRepo sdk.ProjectRepository

	// If no project key in path, get it from workflow run
	if projKey == "" || projKey == ef.currentProject {
		projKey = ef.currentProject
	} else {
		// Verify project read permission
		can, err := rbac.HasRoleOnProjectAndUserID(ctx, db, sdk.ProjectRoleRead, ef.currentUserID, projKey)
		if err != nil {
			return "", "", err
		}
		if !can {
			return "", fmt.Sprintf("user %s do not have the permission to access %s", ef.currentUserName, name), nil
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
					return "", fmt.Sprintf("vcs %s not found on project %s", vcsName, projKey), nil
				}
				return "", "", err
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
					return "", fmt.Sprintf("repository %s not found on vcs %s into project %s", repoName, vcsName, projKey), nil
				}
				return "", "", err
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
					return "", "", err
				}
				b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					return "", "", err
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
			return "", "", err
		}
		b, err := client.Branch(ctx, entityRepo.Name, sdk.VCSBranchFilters{BranchName: branchOrTag})
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return "", "", err
		}

		if b == nil {
			// try to get tag
			t, err := client.Tag(ctx, entityRepo.Name, branchOrTag)
			if err != nil {
				return "", "", err
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
		if _, has := ef.actionsCache[completePath]; has {
			return completePath, "", nil
		}
	case sdk.EntityTypeWorkerModel:
		if _, has := ef.workerModelCache[completePath]; has {
			return completePath, "", nil
		}
	}

	var entityDB *sdk.Entity
	var err error
	if projKey != ef.currentProject || entityVCS.Name != ef.currentVCS.Name || entityRepo.Name != ef.currentRepo.Name || ref != ef.currentRef {
		entityDB, err = entity.LoadByRefTypeNameCommit(ctx, db, entityRepo.ID, ref, entityType, entityName, "HEAD")
	} else {
		entityDB, err = entity.LoadByRefTypeNameCommit(ctx, db, entityRepo.ID, ef.currentRef, entityType, entityName, ef.currentSha)
	}
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return "", fmt.Sprintf("unable to find workflow dependency: %s", name), nil
		}
		return "", "", err
	}
	switch entityType {
	case sdk.EntityTypeAction:
		var act sdk.V2Action
		if err := yaml.Unmarshal([]byte(entityDB.Data), &act); err != nil {
			return "", "", err
		}
		ef.actionsCache[completePath] = act
	case sdk.EntityTypeWorkerModel:
		var wm sdk.V2WorkerModel
		if err := yaml.Unmarshal([]byte(entityDB.Data), &wm); err != nil {
			return "", "", err
		}
		ef.workerModelCache[completePath] = wm
	case sdk.EntityTypeWorkflowTemplate:
		var wt sdk.V2WorkflowTemplate
		if err := yaml.Unmarshal([]byte(entityDB.Data), &wt); err != nil {
			return "", "", err
		}
		ef.templatesCache[completePath] = sdk.EntityWithObject{
			Entity:   *entityDB,
			Template: wt,
		}
	default:
		return "", "", sdk.NewErrorFrom(sdk.ErrNotImplemented, "entity %s not implemented", entityType)
	}
	return completePath, "", nil
}
