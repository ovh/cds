package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/venom"

	"github.com/golang/protobuf/ptypes"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) updateStepStatusHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		buildID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateStepStatusHandler> Invalid id")
		}

		pbJob, errJob := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, buildID)
		if errJob != nil {
			return sdk.WrapError(errJob, "updateStepStatusHandler> Cannot get pipeline build job %d", buildID)
		}

		var step sdk.StepStatus
		if err := UnmarshalBody(r, &step); err != nil {
			return sdk.WrapError(err, "updateStepStatusHandler> Error while unmarshal job")
		}

		found := false
		for i := range pbJob.Job.StepStatus {
			jobStep := &pbJob.Job.StepStatus[i]
			if step.StepOrder == jobStep.StepOrder {
				jobStep.Status = step.Status
				found = true
			}
		}
		if !found {
			pbJob.Job.StepStatus = append(pbJob.Job.StepStatus, step)
		}

		if err := pipeline.UpdatePipelineBuildJob(api.mustDB(), pbJob); err != nil {
			return sdk.WrapError(err, "updateStepStatusHandler> Cannot update pipeline build job")
		}
		return nil
	}
}

func (api *API) getPipelineBuildTriggeredHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")

		buildNumber, err := requestVarInt(r, "build")
		if err != nil {
			return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> invalid build number")
		}

		// Load Pipeline
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load pipeline %s", pipelineName)
		}

		// Load Application
		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load application %s", appName)
		}

		// Load Env
		env := &sdk.DefaultEnv
		if envName != sdk.DefaultEnv.Name && envName != "" {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load environment %s", envName)
			}
		}

		// Load Children
		pbs, err := pipeline.LoadPipelineBuildChildren(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoPipelineBuild, "getPipelineBuildTriggeredHandler> Cannot load pipeline build children: %s", err)
		}
		return WriteJSON(w, r, pbs, http.StatusOK)
	}
}

func (api *API) deleteBuildHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")

		buildNumber, err := requestVarInt(r, "build")
		if err != nil {
			return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> invalid build number")
		}

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "deleteBuildHandler> Cannot load pipeline %s", pipelineName)
		}

		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteBuildHandler> Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "deleteBuildHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "deleteBuildHandler> No enought right on this environment %s", envName)
		}

		pbID, errPB := pipeline.LoadPipelineBuildID(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errPB != nil {
			return sdk.WrapError(errPB, "deleteBuildHandler> Cannot load pipeline build")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteBuildHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeletePipelineBuildByID(tx, pbID); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "deleteBuildHandler> Cannot delete pipeline build: %s", err)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteBuildHandler> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) getBuildStateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")
		withArtifacts := r.FormValue("withArtifacts")
		withTests := r.FormValue("withTests")

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getBuildStateHandler> Cannot load pipeline %s", pipelineName)
		}

		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getBuildStateHandler> Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "getBuildStateHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getBuildStateHandler> No enought right on this environment %s: ", envName)
		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			lastBuildNumber, errg := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errg != nil {
				return sdk.WrapError(sdk.ErrNotFound, "getBuildStateHandler> Cannot load last pipeline build number for %s-%s-%s: %s", a.Name, pipelineName, env.Name, errg)
			}
			buildNumber = lastBuildNumber
		} else {
			buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
			if err != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "getBuildStateHandler> Cannot parse build number %s: %s", buildNumberS, err)
			}
		}

		// load pipeline_build.id
		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if err != nil {
			return sdk.WrapError(err, "getBuildStateHandler> %s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d)", getUser(ctx).Username, projectKey, appName, pipelineName, env.Name, buildNumber)
		}

		if withArtifacts == "true" {
			var errLoadArtifact error
			pb.Artifacts, errLoadArtifact = artifact.LoadArtifactsByBuildNumber(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
			if errLoadArtifact != nil {
				return sdk.WrapError(errLoadArtifact, "getBuildStateHandler> Cannot load artifacts")
			}
		}

		if withTests == "true" {
			tests, errLoadTests := pipeline.LoadTestResults(api.mustDB(), pb.ID)
			if errLoadTests != nil {
				return sdk.WrapError(errLoadTests, "getBuildStateHandler> Cannot load tests")
			}
			if len(tests.TestSuites) > 0 {
				pb.Tests = &tests
			}
		}
		pb.Translate(r.Header.Get("Accept-Language"))

		return WriteJSON(w, r, pb, http.StatusOK)
	}
}

func (api *API) addQueueResultHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "addQueueResultHandler> invalid id")
		}

		// Load Build
		pbJob, errJob := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, id)
		if errJob != nil {
			return sdk.WrapError(sdk.ErrNotFound, "addQueueResultHandler> Cannot load queue (%d) from db: %s", id, errJob)
		}

		// Unmarshal into results
		var res sdk.Result
		if err := UnmarshalBody(r, &res); err != nil {
			return sdk.WrapError(err, "addQueueResultHandler> cannot unmarshal request")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "addQueueResultHandler> Cannot begin tx")
		}
		defer tx.Rollback()

		//Update worker status
		if err := worker.UpdateWorkerStatus(tx, getWorker(ctx).ID, sdk.StatusWaiting); err != nil {
			log.Warning("addQueueResultHandler> Cannot update worker status (%s): %s", getWorker(ctx).ID, err)
			// We want to update pipelineBuildJob status anyway
		}

		// Update action status
		log.Debug("addQueueResultHandler> Updating %d to %s in queue", id, res.Status)
		if err := pipeline.UpdatePipelineBuildJobStatus(tx, pbJob, sdk.Status(res.Status)); err != nil {
			return sdk.WrapError(err, "addQueueResultHandler> Cannot update %d status", id)
		}

		remoteTime, errt := ptypes.Timestamp(res.RemoteTime)
		if errt != nil {
			return sdk.WrapError(errt, "addQueueResultHandler> Cannot parse remote time")
		}

		infos := []sdk.SpawnInfo{{
			RemoteTime: remoteTime,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{getWorker(ctx).Name, res.Duration}},
		}}

		if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, api.Cache, pbJob.ID, infos); err != nil {
			log.Error("addQueueResultHandler> Cannot save spawn info job %d: %s", pbJob.ID, err)
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addQueueResultHandler> Cannot commit tx")
		}

		return nil
	}
}

func (api *API) getPipelineBuildJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "getPipelineBuildJobHandler> invalid id")
		}

		j, err := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, id)
		if err != nil {
			if err == sql.ErrNoRows {
				err = sdk.ErrPipelineBuildNotFound
			}
			return sdk.WrapError(err, "getPipelineBuildJobHandler> Unable to load pipeline build job id")
		}
		return WriteJSON(w, r, j, http.StatusOK)
	}
}

func (api *API) takePipelineBuildJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "takePipelineBuildJobHandler> invalid id")
		}

		takeForm := &worker.TakeForm{}
		if err := UnmarshalBody(r, takeForm); err != nil {
			return sdk.WrapError(err, "takePipelineBuildJobHandler> cannot unmarshal request")
		}

		// Load worker
		caller := getWorker(ctx)
		if caller.Status != sdk.StatusChecking {
			return sdk.WrapError(sdk.ErrWrongRequest, "takePipelineBuildJobHandler> worker %s is not available to for build (status = %s)", caller.Name, caller.Status)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "takePipelineBuildJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		workerModel := caller.Name
		if caller.ModelID != 0 {
			wm, errModel := worker.LoadWorkerModelByID(api.mustDB(), caller.ModelID)
			if errModel != nil {
				return sdk.ErrNoWorkerModel
			}
			workerModel = wm.Name
		}

		infos := []sdk.SpawnInfo{{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{getWorker(ctx).Name}},
		}}

		if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
			infos = append(infos, sdk.SpawnInfo{
				RemoteTime: takeForm.Time,
				Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJob.ID, Args: []interface{}{getWorker(ctx).Name}},
			})
		}

		pbJob, errTake := pipeline.TakePipelineBuildJob(tx, api.Cache, id, workerModel, caller.Name, infos)
		if errTake != nil {
			return sdk.WrapError(errTake, "takePipelineBuildJobHandler> Cannot take job %d", id)
		}

		if err := worker.SetToBuilding(tx, getWorker(ctx).ID, pbJob.ID, sdk.JobTypePipeline); err != nil {
			return sdk.WrapError(err, "takePipelineBuildJobHandler> Cannot update worker status")
		}

		pbji := worker.PipelineBuildJobInfo{}
		pb, errPb := pipeline.LoadPipelineBuildByID(api.mustDB(), pbJob.PipelineBuildID)
		if errPb != nil {
			return sdk.WrapError(errPb, "takePipelineBuildJobHandler> Cannot get pipeline build")
		}
		pbji.PipelineBuildJob = *pbJob
		pbji.PipelineID = pb.Pipeline.ID
		pbji.BuildNumber = pb.BuildNumber

		if errSecret := loadActionBuildSecretsAndKeys(api.mustDB(), api.Cache, pbJob.ID, &pbji); errSecret != nil {
			return sdk.WrapError(errSecret, "takePipelineBuildJobHandler> Cannot load action build secrets")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "takePipelineBuildJobHandler> Cannot commit transaction")
		}
		return WriteJSON(w, r, pbji, http.StatusOK)
	}
}

func (api *API) bookPipelineBuildJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "bookPipelineBuildJobHandler> invalid id")
		}

		if _, err := pipeline.BookPipelineBuildJob(api.Cache, id, getHatchery(ctx)); err != nil {
			return sdk.WrapError(err, "bookPipelineBuildJobHandler> job already booked")
		}
		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

func (api *API) addSpawnInfosPipelineBuildJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		pbJobID, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "addSpawnInfosPipelineBuildJobHandler> invalid id")
		}
		var s []sdk.SpawnInfo
		if err := UnmarshalBody(r, &s); err != nil {
			return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> cannot unmarshal request")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addSpawnInfosPipelineBuildJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, api.Cache, pbJobID, s); err != nil {
			return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot save job %d", pbJobID)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot commit tx")
		}

		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

func loadActionBuildSecretsAndKeys(db *gorp.DbMap, store cache.Store, pbJobID int64, pbji *worker.PipelineBuildJobInfo) error {
	query := `SELECT pipeline.project_id, pipeline_build.application_id, pipeline_build.environment_id
	FROM pipeline_build
	JOIN pipeline_build_job ON pipeline_build_job.pipeline_build_id = pipeline_build.id
	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	WHERE pipeline_build_job.id = $1`

	var projectID, appID, envID int64
	if err := db.QueryRow(query, pbJobID).Scan(&projectID, &appID, &envID); err != nil {
		return err
	}

	if errS := loadActionBuildSecrets(db, store, projectID, appID, envID, pbji); errS != nil {
		return sdk.WrapError(errS, "loadActionBuildSecretsAndKeys> Cannot load secrets")
	}

	if errK := loadActionBuildKeys(db, store, projectID, appID, envID, pbji); errK != nil {
		return sdk.WrapError(errK, "loadActionBuildSecretsAndKeys> Cannot load keys")
	}

	return nil
}

func loadActionBuildKeys(db gorp.SqlExecutor, store cache.Store, projectID, appID, envID int64, pbji *worker.PipelineBuildJobInfo) error {
	p, errP := project.LoadByID(db, store, projectID, nil, project.LoadOptions.WithKeys)
	if errP != nil {
		return sdk.WrapError(errP, "loadActionBuildKeys> Cannot load project keys")
	}
	for _, k := range p.Keys {
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
			Name:  "cds.proj." + k.Name + ".pub",
			Type:  "string",
			Value: k.Public,
		})
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
			Name:  "cds.proj." + k.Name + ".id",
			Type:  "string",
			Value: k.KeyID,
		})
		pbji.Secrets = append(pbji.Secrets, sdk.Variable{
			Name:  "cds.proj." + k.Name + ".priv",
			Type:  "string",
			Value: k.Private,
		})
	}

	a, errA := application.LoadByID(db, store, appID, nil, application.LoadOptions.WithKeys)
	if errA != nil {
		return sdk.WrapError(errA, "loadActionBuildKeys> Cannot load application keys")
	}
	for _, k := range a.Keys {
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
			Name:  "cds.app." + k.Name + ".pub",
			Type:  "string",
			Value: k.Public,
		})
		pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
			Name:  "cds.app." + k.Name + ".id",
			Type:  "string",
			Value: k.KeyID,
		})
		pbji.Secrets = append(pbji.Secrets, sdk.Variable{
			Name:  "cds.app." + k.Name + ".priv",
			Type:  "string",
			Value: k.Private,
		})
	}

	if envID != sdk.DefaultEnv.ID {
		e, errE := environment.LoadEnvironmentByID(db, envID)
		if errE != nil {
			return sdk.WrapError(errE, "loadActionBuildKeys> Cannot load environment keys")
		}
		for _, k := range e.Keys {
			pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
				Name:  "cds.env." + k.Name + ".pub",
				Type:  "string",
				Value: k.Public,
			})
			pbji.PipelineBuildJob.Parameters = append(pbji.PipelineBuildJob.Parameters, sdk.Parameter{
				Name:  "cds.env." + k.Name + ".id",
				Type:  "string",
				Value: k.KeyID,
			})
			pbji.Secrets = append(pbji.Secrets, sdk.Variable{
				Name:  "cds.env." + k.Name + ".priv",
				Type:  "string",
				Value: k.Private,
			})
		}
	}
	return nil
}

func loadActionBuildSecrets(db gorp.SqlExecutor, store cache.Store, projectID, appID, envID int64, pbji *worker.PipelineBuildJobInfo) error {
	var secrets []sdk.Variable
	// Load project secrets
	pv, err := project.GetAllVariableInProject(db, projectID, project.WithClearPassword())
	if err != nil {
		return err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Error("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return fmt.Errorf("Loaded placeholder for %s", s.Name)
		}
		s.Name = "cds.proj." + s.Name
		secrets = append(secrets, s)
	}

	// Load application secrets
	pv, err = application.GetAllVariableByID(db, appID, application.WithClearPassword())
	if err != nil {
		return err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Error("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return fmt.Errorf("Loaded placeholder for %s", s.Name)
		}
		s.Name = "cds.app." + s.Name
		secrets = append(secrets, s)
	}

	// Load environment secrets
	pv, err = environment.GetAllVariableByID(db, envID, environment.WithClearPassword())
	if err != nil {
		return err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Error("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return fmt.Errorf("Loaded placeholder for %s", s.Name)
		}
		s.Name = "cds.env." + s.Name
		secrets = append(secrets, s)
	}
	pbji.Secrets = secrets
	return nil
}

func (api *API) getQueueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var queue []sdk.PipelineBuildJob
		var errQ error
		switch getAgent(r) {
		case sdk.HatcheryAgent:
			queue, errQ = pipeline.LoadGroupWaitingQueue(api.mustDB(), api.Cache, getHatchery(ctx).GroupID)
		case sdk.WorkerAgent:
			queue, errQ = pipeline.LoadGroupWaitingQueue(api.mustDB(), api.Cache, getWorker(ctx).GroupID)
		default:
			queue, errQ = pipeline.LoadUserWaitingQueue(api.mustDB(), api.Cache, getUser(ctx))
		}

		lang := r.Header.Get("Accept-Language")
		for p := range queue {
			queue[p].Translate(lang)
		}

		if errQ != nil {
			return sdk.WrapError(errQ, "getQueueHandler> Cannot load queue from db: %s", errQ)
		}

		return WriteJSON(w, r, queue, http.StatusOK)
	}
}

func (api *API) addBuildVariableHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["app"]

		buildNumber, errInt := requestVarInt(r, "build")
		if errInt != nil {
			return sdk.WrapError(errInt, "addBuildTestResultsHandler> invalid build number")
		}

		var env *sdk.Environment
		envName := r.FormValue("envName")
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var err error
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownEnv, "addBuildVariableHandler> Cannot load environment %s: %s", envName, err)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addBuildVariableHandler> No enought right on this environment %s", envName)
		}

		// Check that pipeline exists
		p, errLP := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if errLP != nil {
			return sdk.WrapError(errLP, "addBuildVariableHandler> Cannot load pipeline %s", pipelineName)
		}

		// Check that application exists
		a, errLA := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errLA != nil {
			return sdk.WrapError(errLA, "addBuildVariableHandler> Cannot load application %s", appName)
		}

		// load pipeline_build.id
		pbID, errPB := pipeline.LoadPipelineBuildID(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errPB != nil {
			return sdk.WrapError(errPB, "addBuildVariableHandler> Cannot load pipeline build %d", buildNumber)
		}

		// Unmarshal into results
		var v sdk.Variable
		if err := UnmarshalBody(r, &v); err != nil {
			return sdk.WrapError(err, "addBuildVariableHandler> cannot unmarshal request")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addBuildVariableHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.InsertBuildVariable(tx, api.Cache, pbID, v); err != nil {
			return sdk.WrapError(err, "addBuildVariableHandler> Cannot add build variable")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addBuildVariableHandler> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) addBuildTestResultsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["app"]

		buildNumber, errInt := requestVarInt(r, "build")
		if errInt != nil {
			return sdk.WrapError(errInt, "addBuildTestResultsHandler> invalid build number")
		}

		var env *sdk.Environment
		envName := r.FormValue("envName")
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errle error
			env, errle = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if errle != nil {
				return sdk.WrapError(errle, "addBuildTestResultsHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "addBuildTestResultsHandler> No enought right on this environment %s: ", envName)
		}

		// Check that pipeline exists
		p, errlp := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if errlp != nil {
			return sdk.WrapError(errlp, "addBuildTestResultsHandler> Cannot load pipeline %s", pipelineName)
		}

		// Check that application exists
		a, errln := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if errln != nil {
			return sdk.WrapError(errln, "addBuildTestResultsHandler> Cannot load application %s", appName)
		}

		// load pipeline_build.id
		pb, errl := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errl != nil {
			return sdk.WrapError(errl, "addBuiltTestResultsHandler> Cannot loadpipelinebuild for %s/%s[%s] %d", a.Name, p.Name, envName, buildNumber)
		}

		// Unmarshal into results
		var new venom.Tests
		if err := UnmarshalBody(r, &new); err != nil {
			return sdk.WrapError(err, "addBuildVariableHandler> cannot unmarshal request")
		}

		// Load existing and merge
		tests, err := pipeline.LoadTestResults(api.mustDB(), pb.ID)
		if err != nil {
			return sdk.WrapError(err, "addBuildTestResultsHandler> Cannot load test results")
		}

		for k := range new.TestSuites {
			for i := range tests.TestSuites {
				if tests.TestSuites[i].Name == new.TestSuites[k].Name {
					// testsuite with same name already exists,
					// Create a unique name
					new.TestSuites[k].Name = fmt.Sprintf("%s.%d", new.TestSuites[k].Name, pb.ID)
					break
				}
			}
			tests.TestSuites = append(tests.TestSuites, new.TestSuites[k])
		}

		// update total values
		tests.Total = 0
		tests.TotalOK = 0
		tests.TotalKO = 0
		tests.TotalSkipped = 0
		for _, ts := range tests.TestSuites {
			tests.Total += ts.Total
			tests.TotalKO += ts.Failures + ts.Errors
			tests.TotalOK += ts.Total - ts.Skipped - ts.Failures - ts.Errors
			tests.TotalSkipped += ts.Skipped
		}

		if err := pipeline.UpdateTestResults(api.mustDB(), pb.ID, tests); err != nil {
			return sdk.WrapError(err, "addBuildTestsResultsHandler> Cannot insert tests results")
		}

		stats.TestEvent(api.mustDB(), p.ProjectID, a.ID, tests)
		return nil
	}
}

func (api *API) getBuildTestResultsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		appName := vars["app"]

		var err error
		var env *sdk.Environment
		envName := r.FormValue("envName")
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownEnv, "getBuildTestResultsHandler> Cannot load environment %s: %s", envName, err)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getBuildTestResultsHandler> No enought right on this environment %s: ", envName)
		}

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "getBuildTestResultsHandler> Cannot load pipeline %s", pipelineName)
		}

		// Check that application exists
		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getBuildTestResultsHandler> Cannot load application %s", appName)
		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			var errlb error
			bn, errlb := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errlb != nil {
				return sdk.WrapError(sdk.ErrNoPipelineBuild, "getBuildTestResultsHandler> Cannot load last build number for %s: %s", pipelineName, errlb)
			}
			buildNumber = bn
		} else {
			var errpi error
			buildNumber, errpi = strconv.ParseInt(buildNumberS, 10, 64)
			if errpi != nil {
				return sdk.WrapError(errpi, "getBuildTestResultsHandler> Cannot parse build number %s", buildNumberS)
			}
		}

		// load pipeline_build.id
		pb, errlpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errlpb != nil {
			return sdk.WrapError(errlpb, "getBuildTestResultsHandler> Cannot load pipeline build")
		}

		tests, errltr := pipeline.LoadTestResults(api.mustDB(), pb.ID)
		if errltr != nil {
			return sdk.WrapError(errltr, "getBuildTestResultsHandler> Cannot load test results")
		}

		return WriteJSON(w, r, tests, http.StatusOK)
	}
}
