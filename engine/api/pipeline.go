package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/service"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadDestEnvFromRunRequest(ctx context.Context, db *gorp.DbMap, request *sdk.RunRequest, projectKey string) (*sdk.Environment, error) {
	var envDest = &sdk.DefaultEnv
	var err error
	if request.Env.Name != "" && request.Env.Name != sdk.DefaultEnv.Name {
		envDest, err = environment.LoadEnvironmentByName(db, projectKey, request.Env.Name)
		if err != nil {
			log.Warning("loadDestEnvFromRunRequest> Cannot load destination environmens %s: %v", request.Env.Name, err)
			return nil, sdk.ErrNoEnvironment
		}
	}
	if !permission.AccessToEnvironment(projectKey, envDest.Name, getUser(ctx), permission.PermissionReadExecute) {
		log.Warning("loadDestEnvFromRunRequest> You do not have Execution Right on this environment %s", envDest.Name)
		return nil, sdk.ErrForbidden
	}
	return envDest, nil
}

func (api *API) runPipelineWithLastParentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		app, errl := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
		if errl != nil {
			if !sdk.ErrorIs(errl, sdk.ErrApplicationNotFound) {
				log.Warning("runPipelineWithLastParentHandler> Cannot load application %s: %s\n", appName, errl)
			}
			return errl
		}

		var request sdk.RunRequest
		if err := service.UnmarshalBody(r, &request); err != nil {
			return err
		}

		//Check parent stuff
		if request.ParentApplicationID == 0 && request.ParentPipelineID == 0 {
			return sdk.ErrParentApplicationAndPipelineMandatory
		}
		envID := sdk.DefaultEnv.ID
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}

		// Load pipeline
		pip, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				log.Warning("runPipelineWithLastParentHandler> Cannot load pipeline %s; %s\n", pipelineName, err)
			}
			return err
		}

		// Check that pipeline is attached to application
		ok, err := application.IsAttached(api.mustDB(), app.ProjectID, app.ID, pip.Name)
		if !ok {
			return sdk.WrapError(sdk.ErrPipelineNotAttached, "runPipelineWithLastParentHandler> Pipeline %s is not attached to app %s", pipelineName, appName)
		}
		if err != nil {
			return sdk.WrapError(err, "Cannot check if pipeline %s is attached to %s", pipelineName, appName)
		}

		//Load environment
		envDest, err := loadDestEnvFromRunRequest(ctx, api.mustDB(), &request, projectKey)
		if err != nil {
			return err
		}

		//Load triggers
		triggers, err := trigger.LoadTriggers(api.mustDB(), app.ID, pip.ID, envDest.ID)
		if err != nil {
			return err
		}

		//Find parent trigger
		var trig *sdk.PipelineTrigger
		for i, t := range triggers {
			log.Debug("Trigger from app(%s[%d]) pip(%s[%d]) env(%s[%d]) to app(%s[%d]) pip(%s[%d]) env(%s[%d])\n", t.SrcApplication.Name, t.SrcApplication.ID, t.SrcPipeline.Name, t.SrcPipeline.ID, t.SrcEnvironment.Name, t.SrcEnvironment.ID, t.DestApplication.Name, t.DestApplication.ID, t.DestPipeline.Name, t.DestPipeline.ID, t.DestEnvironment.Name, t.DestEnvironment.ID)
			if t.SrcApplication.ID == request.ParentApplicationID &&
				t.SrcPipeline.ID == request.ParentPipelineID &&
				t.SrcEnvironment.ID == envID {
				trig = &triggers[i]
			}
		}

		//If trigger not found: exit
		if trig == nil {
			return sdk.ErrPipelineNotFound
		}

		//Branch
		branch, remote := pipeline.GetVCSInfosInParams(request.Params)

		opts := []pipeline.ExecOptionFunc{
			pipeline.LoadPipelineBuildOpts.WithStatus(sdk.StatusSuccess.String()),
			pipeline.LoadPipelineBuildOpts.WithBranchName(branch),
		}

		if remote == "" {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithEmptyRemote(remote))
		} else {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithRemoteName(remote))
		}

		builds, err := pipeline.LoadPipelineBuildsByApplicationAndPipeline(api.mustDB(), request.ParentApplicationID, request.ParentPipelineID, envID, 1, opts...)

		if err != nil {
			return sdk.WrapError(sdk.ErrNoParentBuildFound, "runPipelineWithLastParentHandler> Unable to find any successful pipeline build")
		}

		if len(builds) == 0 {
			return sdk.ErrNoPipelineBuild
		}

		request.ParentBuildNumber = builds[0].BuildNumber

		return api.runPipelineHandlerFunc(ctx, w, r, &request)
	}
}

func (api *API) runPipelineHandlerFunc(ctx context.Context, w http.ResponseWriter, r *http.Request, request *sdk.RunRequest) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	//Load the project
	proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
	if errproj != nil {
		return sdk.WrapError(errproj, "runPipelineHandler> Unable to load project %s", projectKey)
	}

	app, errln := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if errln != nil {
		if !sdk.ErrorIs(errln, sdk.ErrApplicationNotFound) {
			log.Warning("runPipelineHandler> Cannot load application %s: %s\n", appName, errln)
		}
		return errln
	}

	var pip *sdk.Pipeline
	for _, p := range app.Pipelines {
		if p.Pipeline.Name == pipelineName {
			pip = &p.Pipeline
			break
		}
	}

	if pip == nil {
		return sdk.WrapError(sdk.ErrPipelineNotAttached, "runPipelineHandler> Pipeline %s is not attached to app %s", pipelineName, appName)
	}

	version := int64(0)
	// Load parent pipeline build + add parent variable
	var parentPipelineBuild *sdk.PipelineBuild
	envID := sdk.DefaultEnv.ID
	if request.ParentBuildNumber != 0 {
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}
		pb, errlp := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), request.ParentApplicationID, request.ParentPipelineID, envID, request.ParentBuildNumber)
		if errlp != nil {
			return sdk.WrapError(errlp, "runPipelineHandler> Cannot load parent pipeline build")
		}
		parentParams := queue.ParentBuildInfos(pb)
		request.Params = append(request.Params, parentParams...)

		version = pb.Version
		parentPipelineBuild = pb
	} else if request.ParentVersion != 0 {
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}

		pbs, errP := pipeline.LoadPipelineBuildByApplicationPipelineEnvVersion(api.mustDB(), request.ParentApplicationID, request.ParentPipelineID, envID, request.ParentVersion, 1)
		if errP != nil {
			return sdk.WrapError(errP, "runPipelineHandler> Cannot load parent pipeline build by version")
		}
		if len(pbs) == 0 {
			return sdk.WrapError(sdk.ErrNoParentBuildFound, "runPipelineHandler> No parent build found")
		}
		parentParams := queue.ParentBuildInfos(&pbs[0])
		request.Params = append(request.Params, parentParams...)

		version = pbs[0].Version
		parentPipelineBuild = &pbs[0]
	}

	envDest, err := loadDestEnvFromRunRequest(ctx, api.mustDB(), request, projectKey)
	if err != nil {
		return sdk.WrapError(err, "Unable to load dest environment")
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WrapError(err, "Cannot start tx")
	}
	defer tx.Rollback()

	// Schedule pipeline for build
	log.Debug("runPipelineHandler> Scheduling %s/%s/%s[%s] with %d params, version 0",
		projectKey, app.Name, pipelineName, envDest.Name, len(request.Params))
	log.Debug("runPipelineHandler> Pipeline trigger by %d - %d", getUser(ctx).ID, request.ParentPipelineID)
	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger:       true,
		TriggeredBy:         getUser(ctx),
		ParentPipelineBuild: parentPipelineBuild,
	}
	if parentPipelineBuild != nil {
		trigger.VCSChangesAuthor = parentPipelineBuild.Trigger.VCSChangesAuthor
		trigger.VCSChangesHash = parentPipelineBuild.Trigger.VCSChangesHash
		trigger.VCSChangesBranch = parentPipelineBuild.Trigger.VCSChangesBranch
		trigger.VCSRemote = parentPipelineBuild.Trigger.VCSRemote
		trigger.VCSRemoteURL = parentPipelineBuild.Trigger.VCSRemoteURL
	}

	env, errenv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, envDest.Name)
	if errenv != nil {
		return sdk.WrapError(errenv, "runPipelineHandler> Unable to load env %s %s", projectKey, envDest.Name)
	}

	pb, err := queue.RunPipeline(api.mustDB, api.Cache, tx, projectKey, app, pipelineName, envDest.Name, request.Params, version, trigger, getUser(ctx))
	if err != nil {
		return sdk.WrapError(err, "Cannot run pipeline")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Cannot commit tx")
	}

	go func() {
		if _, err := pipeline.UpdatePipelineBuildCommits(api.mustDB(), api.Cache, proj, pip, app, env, pb); err != nil {
			log.Warning("runPipelineHandler> Unable to update pipeline build commits : %s", err)
		}
	}()

	return service.WriteJSON(w, pb, http.StatusOK)
}

func (api *API) runPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var request sdk.RunRequest
		if err := service.UnmarshalBody(r, &request); err != nil {
			return err
		}

		return api.runPipelineHandlerFunc(ctx, w, r, &request)
	}
}

func (api *API) updatePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permPipelineKey"]

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		// check pipeline name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "updatePipelineHandler: Pipeline name %s do not respect pattern", p.Name)
		}

		pipelineDB, err := pipeline.LoadPipeline(api.mustDB(), key, name, true)
		if err != nil {
			return sdk.WrapError(err, "cannot load pipeline %s", name)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			sdk.WrapError(errB, "updatePipelineHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.CreateAudit(tx, pipelineDB, pipeline.AuditUpdatePipeline, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot create audit")
		}

		oldName := pipelineDB.Name
		pipelineDB.Name = p.Name
		pipelineDB.Description = p.Description
		pipelineDB.Type = p.Type

		if err := pipeline.UpdatePipeline(tx, pipelineDB); err != nil {
			return sdk.WrapError(err, "cannot update pipeline %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(key, p.Name, oldName, getUser(ctx))

		return service.WriteJSON(w, pipelineDB, http.StatusOK)
	}
}

func (api *API) postPipelineRollbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permPipelineKey"]
		auditID, errConv := strconv.ParseInt(vars["auditID"], 10, 64)
		if errConv != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postPipelineRollbackHandler> cannot convert auditID to int")
		}

		db := api.mustDB()
		u := getUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u)
		if errP != nil {
			return sdk.WrapError(errP, "postPipelineRollbackHandler> Cannot load project")
		}

		audit, errA := pipeline.LoadAuditByID(db, auditID)
		if errA != nil {
			return sdk.WrapError(errA, "postPipelineRollbackHandler> Cannot load audit %d", auditID)
		}

		if err := pipeline.LoadGroupByPipeline(ctx, db, audit.Pipeline); err != nil {
			return sdk.WrapError(err, "cannot load group by pipeline")
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "postPipelineRollbackHandler> cannot begin transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		done := new(sync.WaitGroup)
		done.Add(1)
		msgChan := make(chan sdk.Message)
		msgList := []sdk.Message{}
		go func(array *[]sdk.Message) {
			defer done.Done()
			for m := range msgChan {
				*array = append(*array, m)
			}
		}(&msgList)

		if err := pipeline.ImportUpdate(tx, proj, audit.Pipeline, msgChan, u); err != nil {
			return sdk.WrapError(err, "cannot import pipeline")
		}

		close(msgChan)
		done.Wait()

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineUpdate(key, audit.Pipeline.Name, name, u)

		return service.WriteJSON(w, *audit.Pipeline, http.StatusOK)
	}
}

func (api *API) getApplicationUsingPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permPipelineKey"]

		pipelineData, err := pipeline.LoadPipeline(api.mustDB(), key, name, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", name)
		}
		applications, err := application.LoadByPipeline(api.mustDB(), api.Cache, pipelineData.ID, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load applications using pipeline %s", name)
		}

		return service.WriteJSON(w, applications, http.StatusOK)
	}
}

func (api *API) addPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errl := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errl != nil {
			return sdk.WrapError(errl, "AddPipeline: Cannot load %s", key)
		}

		var p sdk.Pipeline
		if err := service.UnmarshalBody(r, &p); err != nil {
			return err
		}

		// check pipeline name pattern
		if regexp := sdk.NamePatternRegex; !regexp.MatchString(p.Name) {
			return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "AddPipeline: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		}

		// Check that pipeline does not already exists
		exist, err := pipeline.ExistPipeline(api.mustDB(), proj.ID, p.Name)
		if err != nil {
			return sdk.WrapError(err, "cannot check if pipeline exist")
		}
		if exist {
			return sdk.WrapError(sdk.ErrConflict, "addPipeline> Pipeline %s already exists", p.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()

		p.ProjectID = proj.ID
		if err := pipeline.InsertPipeline(tx, api.Cache, proj, &p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert pipeline")
		}

		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "Cannot load groupfrom project")
		}

		for _, g := range proj.ProjectGroups {
			p.GroupPermission = append(p.GroupPermission, g)
		}

		if err := group.InsertGroupsInPipeline(tx, proj.ProjectGroups, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot add groups on pipeline")
		}

		if p.Usage != nil {
			for _, app := range p.Usage.Applications {
				if _, err := application.AttachPipeline(tx, app.ID, p.ID); err != nil {
					return sdk.WrapError(err, "Cannot attach pipeline %d to %d", app.ID, p.ID)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineAdd(key, p, getUser(ctx))

		p.Permission = permission.PermissionReadWriteExecute

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelineAuditHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]

		audits, err := pipeline.LoadAudit(api.mustDB(), projectKey, pipelineName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline audit")
		}
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		withApp := FormBool(r, "withApplications")
		withWorkflows := FormBool(r, "withWorkflows")
		withEnvironments := FormBool(r, "withEnvironments")

		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, true)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		p.Permission = permission.PipelinePermission(projectKey, p.Name, getUser(ctx))

		if withApp || withWorkflows || withEnvironments {
			p.Usage = &sdk.Usage{}
		}

		if withApp {
			apps, errA := application.LoadByPipeline(api.mustDB(), api.Cache, p.ID, getUser(ctx))
			if errA != nil {
				return sdk.WrapError(errA, "getPipelineHandler> Cannot load applications using pipeline %s", p.Name)
			}
			p.Usage.Applications = apps
		}

		if withWorkflows {
			wf, errW := workflow.LoadByPipelineName(api.mustDB(), projectKey, pipelineName)
			if errW != nil {
				return sdk.WrapError(errW, "getPipelineHandler> Cannot load workflows using pipeline %s", p.Name)
			}
			p.Usage.Workflows = wf
		}

		if withEnvironments {
			envs, errE := environment.LoadByPipelineName(api.mustDB(), projectKey, pipelineName)
			if errE != nil {
				return sdk.WrapError(errE, "getPipelineHandler> Cannot load environments using pipeline %s", p.Name)
			}
			p.Usage.Environments = envs
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getPipelineTypeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AvailablePipelineType, http.StatusOK)
	}
}

func (api *API) getPipelinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		project, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNoProject) {
				log.Warning("getPipelinesHandler: Cannot load %s: %s\n", key, err)
			}
			return err
		}

		pip, err := pipeline.LoadPipelines(api.mustDB(), project.ID, true, getUser(ctx))
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) {
				log.Warning("getPipelinesHandler>Cannot load pipelines: %s\n", err)
			}
			return err
		}

		return service.WriteJSON(w, pip, http.StatusOK)
	}
}

func (api *API) getPipelineHistoryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(err, "Cannot parse form")
		}
		envName := r.Form.Get("envName")
		limitString := r.Form.Get("limit")
		status := r.Form.Get("status")
		branchName := r.Form.Get("branchName")
		remote := r.Form.Get("remote")
		buildNumber := r.Form.Get("buildNumber")

		var limit int
		if limitString != "" {
			var erra error
			limit, erra = strconv.Atoi(limitString)
			if erra != nil {
				return erra
			}
		} else {
			limit = 20
		}

		a, errln := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.WithPipelines)
		if errln != nil {
			return sdk.WrapError(errln, "getPipelineHistoryHandler> Cannot load application %s", appName)
		}

		var p *sdk.Pipeline
		for _, apip := range a.Pipelines {
			if apip.Pipeline.Name == pipelineName {
				p = &apip.Pipeline
				break
			}
		}

		if p == nil {
			return sdk.WrapError(sdk.ErrPipelineNotAttached, "Pipeline not found on application")
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errle error
			env, errle = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if errle != nil {
				return sdk.WrapError(errle, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		opts := []pipeline.ExecOptionFunc{
			pipeline.LoadPipelineBuildOpts.WithStatus(status),
			pipeline.LoadPipelineBuildOpts.WithBranchName(branchName),
		}

		if a.RepositoryFullname != "" && (remote == "" || remote == a.RepositoryFullname) {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithEmptyRemote(a.RepositoryFullname))
		} else if remote == "" {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithEmptyRemote(remote))
		} else {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithRemoteName(remote))
		}

		if buildNumber != "" {
			opts = append(opts, pipeline.LoadPipelineBuildOpts.WithBuildNumber(buildNumber))
		}

		pbs, errl := pipeline.LoadPipelineBuildsByApplicationAndPipeline(api.mustDB(), a.ID, p.ID, env.ID, limit, opts...)

		if errl != nil {
			return sdk.WrapError(errl, "Cannot load pipeline %s history", p.Name)
		}

		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) deletePipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load project")
		}

		p, err := pipeline.LoadPipeline(api.mustDB(), proj.Key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		used, err := application.CountPipeline(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if pipeline is used by an application")
		}

		if used {
			return sdk.WrapError(sdk.ErrPipelineHasApplication, "Cannot delete a pipeline used by at least 1 application")
		}

		usedW, err := workflow.CountPipeline(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if pipeline is used by a workflow")
		}

		if usedW {
			return sdk.WrapError(sdk.ErrPipelineUsedByWorkflow, "Cannot delete a pipeline used by at least 1 workflow")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeleteAudit(tx, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline audit")
		}

		if err := pipeline.DeletePipeline(tx, p.ID, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "Cannot delete pipeline %s", pipelineName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishPipelineDelete(key, *p, getUser(ctx))
		return nil
	}
}

func (api *API) getBuildingPipelinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var err error
		var pbs, recent []sdk.PipelineBuild

		if getUser(ctx).Admin {
			recent, err = pipeline.LoadRecentPipelineBuild(api.mustDB())
		} else {
			recent, err = pipeline.LoadUserRecentPipelineBuild(api.mustDB(), getUser(ctx).ID)
		}
		if err != nil {
			return sdk.WrapError(err, "cannot load recent pipelines")

		}
		pbs = append(pbs, recent...)
		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) getPipelineBuildingCommitHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hash := vars["hash"]

		pbs, err := pipeline.LoadPipelineBuildByHash(api.mustDB(), hash)
		if err != nil {
			return err

		}

		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) stopPipelineBuildHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		pipName := vars["permPipelineKey"]

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "stopPipelineBuildHandler> Cannot parse form")
		}
		envName := r.Form.Get("envName")

		buildNumber, err := requestVarInt(r, "build")
		if err != nil {
			return sdk.WrapError(err, "invalid build number")
		}

		pip, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline")
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application")
		}

		if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
			return sdk.ErrNoEnvironmentProvided
		}
		env := &sdk.DefaultEnv

		if pip.Type != sdk.BuildPipeline {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "You do not have Execution Right on this environment %s", env.Name)
		}

		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), app.ID, pip.ID, env.ID, buildNumber)
		if err != nil {
			errFinal := err
			if sdk.ErrorIs(err, sdk.ErrNoPipelineBuild) {
				errFinal = sdk.ErrBuildArchived
			}
			return sdk.WrapError(errFinal, "Cannot load pipeline Build")
		}

		if err := pipeline.StopPipelineBuild(api.mustDB(), api.Cache, pb); err != nil {
			return sdk.WrapError(err, "Cannot stop pipeline build")
		}

		return nil
	}
}

func (api *API) restartPipelineBuildHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		pipName := vars["permPipelineKey"]
		buildNumberS := vars["build"]

		err := r.ParseForm()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "restartPipelineBuildHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")

		buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "restartPipelineBuildHandler> buildNumber is not a int")

		}

		// Load pipeline
		pip, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline")

		}

		// Load application
		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application")

		}

		// Load environment
		if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
			return sdk.ErrNoEnvironmentProvided

		}
		env := &sdk.DefaultEnv

		if pip.Type != sdk.BuildPipeline {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)

			}

			if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s: ", envName)

			}
		}

		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), app.ID, pip.ID, env.ID, buildNumber)
		if err != nil {
			errFinal := err
			if sdk.ErrorIs(err, sdk.ErrNoPipelineBuild) {
				errFinal = sdk.ErrBuildArchived
			}
			return sdk.WrapError(errFinal, "Cannot load pipeline Build")
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrNoEnvExecution, "You do not have Execution Right on this environment %s", env.Name)
		}

		tx, errbegin := api.mustDB().Begin()
		if errbegin != nil {
			return sdk.WrapError(sdk.ErrNoEnvExecution, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.RestartPipelineBuild(tx, pb); err != nil {
			return sdk.WrapError(err, "cannot restart pb")

		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(sdk.ErrNoEnvExecution, "restartPipelineBuildHandler> Cannot commit transaction")

		}

		return service.WriteJSON(w, pb, http.StatusOK)
	}
}

func (api *API) getPipelineCommitsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		pipName := vars["permPipelineKey"]

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "getPipelineCommitsHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")
		hash := r.Form.Get("hash")

		// Load project
		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "getPipelineCommitsHandler> Cannot load project")
		}

		// Load pipeline
		pip, errpip := pipeline.LoadPipeline(api.mustDB(), projectKey, pipName, false)
		if errpip != nil {
			return sdk.WrapError(errpip, "getPipelineCommitsHandler> Cannot load pipeline")
		}

		//Load the environment
		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var err error
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getPipelineCommitsHandler> No enough right on this environment %s (user=%s)", envName, getUser(ctx).Username)
		}

		//Load the application
		app, errapp := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errapp != nil {
			return sdk.WrapError(errapp, "getPipelineCommitsHandler> Unable to load application %s", appName)
		}

		commits := []sdk.VCSCommit{}

		//Check it the application is attached to a repository
		if app.VCSServer == "" {
			log.Warning("getPipelineCommitsHandler> Application %s/%s not attached to a repository manager", projectKey, appName)
			return service.WriteJSON(w, commits, http.StatusOK)
		}

		pbs, errpb := pipeline.LoadPipelineBuildsByApplicationAndPipeline(api.mustDB(), app.ID, pip.ID, env.ID, 1, pipeline.LoadPipelineBuildOpts.WithStatus(string(sdk.StatusSuccess)))
		if errpb != nil {
			return sdk.WrapError(errpb, "getPipelineCommitsHandler> Cannot load pipeline build")
		}

		if len(pbs) != 1 {
			log.Debug("getPipelineCommitsHandler> There is no previous build")
			return service.WriteJSON(w, commits, http.StatusOK)
		}

		if app.RepositoryFullname == "" {
			log.Debug("getPipelineCommitsHandler> No repository on the application %s", appName)
			return service.WriteJSON(w, commits, http.StatusOK)
		}

		//Get the RepositoriesManager Client
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, app.VCSServer)
		client, errclient := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
		if errclient != nil {
			return sdk.WrapError(errclient, "getPipelineCommitsHandler> Cannot get client")
		}

		if pbs[0].Trigger.VCSChangesHash == "" {
			log.Debug("getPipelineCommitsHandler>No hash on the previous run %d", pbs[0].ID)
			return service.WriteJSON(w, commits, http.StatusOK)
		}

		//If we are lucky, return a true diff
		var errcommits error
		commits, errcommits = client.Commits(ctx, app.RepositoryFullname, pbs[0].Trigger.VCSChangesBranch, pbs[0].Trigger.VCSChangesHash, hash)
		if errcommits != nil {
			return sdk.WrapError(errcommits, "getPipelineBuildCommitsHandler> Cannot get commits")
		}

		return service.WriteJSON(w, commits, http.StatusOK)
	}
}

func (api *API) getPipelineBuildCommitsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]
		pipName := vars["permPipelineKey"]
		buildNumber, errS := strconv.Atoi(vars["build"])
		if errS != nil {
			return sdk.ErrInvalidID
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "getPipelineBuildCommitsHandler> Unable to load project %s", projectKey)
		}

		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "getPipelineBuildCommitsHandler> Cannot parse form")

		}
		envName := r.Form.Get("envName")

		// Load pipeline
		pip, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline")

		}

		//Load the environment
		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				if !sdk.ErrorIs(err, sdk.ErrNoEnvironment) {
					log.Warning("getPipelineBuildCommitsHandler> Cannot load environment %s: %s\n", envName, err)
				}
				return err
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getPipelineHistoryHandler> No enough right on this environment %s: ", envName)
		}

		//Load the application
		app, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.ErrApplicationNotFound
		}

		//Load the pipeline build
		pb, errpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), app.ID, pip.ID, env.ID, int64(buildNumber))
		if errpb != nil {
			return sdk.WrapError(errpb, "getPipelineBuildCommitsHandler>")
		}

		//Check it the application is attached to a repository
		if app.VCSServer == "" || app.RepositoryFullname == "" {
			return sdk.ErrNoReposManagerClientAuth
		}

		cm, err := pipeline.UpdatePipelineBuildCommits(api.mustDB(), api.Cache, proj, pip, app, env, pb)
		if err != nil {
			return sdk.WrapError(err, "UpdatePipelineBuildCommits failed")
		}
		return service.WriteJSON(w, cm, http.StatusOK)
	}
}
