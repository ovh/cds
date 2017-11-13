package poller

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Executer is the goroutine which run the pipelines
func Executer(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	tick := time.NewTicker(5 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting poller.Executer: %v", c.Err())
				return
			}
		case <-tick:
			exs, err := ExecuterRun(DBFunc(), store)
			if err != nil {
				log.Warning("poller.Executer> Error : %s", err)
				continue
			}
			if len(exs) > 0 {
				log.Debug("poller.Executer> %d has been executed", len(exs))
			}
		}
	}
}

//ExecuterRun is the core function of Executer goroutine
func ExecuterRun(db *gorp.DbMap, store cache.Store) ([]sdk.RepositoryPollerExecution, error) {
	//Load pending executions
	exs, err := LoadPendingExecutions(db)
	if err != nil {
		return nil, sdk.WrapError(err, "poller.ExecuterRun> Unable to load pending execution")
	}

	//Process all
	for i := range exs {
		go executerRun(db, store, &exs[i])
		time.Sleep(10 * time.Second)
	}

	return exs, nil
}

func executerRun(db *gorp.DbMap, store cache.Store, e *sdk.RepositoryPollerExecution) {
	tx, errb := db.Begin()
	if errb != nil {
		log.Error("poller.ExecuterRun> %s", errb)
		return
	}

	defer tx.Rollback()

	if err := LockPollerExecution(tx, e.ID); err != nil {
		log.Debug("poller.ExecuterRun> LockPollerExecution %d: %s", e.ID, err)
		return
	}

	p, errl := LoadByApplicationAndPipeline(tx, e.ApplicationID, e.PipelineID)
	if errl != nil {
		//If the poller doesn't exist: clean this execution and exit
		if errl == sql.ErrNoRows {
			if err := DeleteExecution(tx, e); err != nil {
				log.Error("poller.ExecuterRun> Unable to delete execution %d: %s", e.ID, err)
			}
			if err := tx.Commit(); err != nil {
				log.Error("poller.ExecuterRun> %s", err)
			}
			return
		}
		log.Error("poller.ExecuterRun> Unable to load poller appID=%d pipID=%d: %s", e.ApplicationID, e.PipelineID, errl)
		return
	}
	pbs, err := executerProcess(tx, store, p, e)
	if err != nil {
		log.Error("poller.ExecuterRun> Unable to process %+v : %s", e, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("poller.ExecuterRun> %s", err)
		return
	}

	//Update pipeline build commits
	app, errapp := application.LoadByID(db, store, e.ApplicationID, nil)
	if errapp != nil {
		log.Warning("poller.ExecuterRun> Unable to load application : %s", errapp)
		return
	}

	proj, errproj := project.Load(db, store, app.ProjectKey, nil)
	if errproj != nil {
		log.Warning("poller.ExecuterRun> Unable to load project : %s", errproj)
		return
	}

	pip, errpip := pipeline.LoadPipelineByID(db, e.PipelineID, true)
	if errpip != nil {
		log.Warning("poller.ExecuterRun> Unable to load pipeline : %s", errpip)
		return
	}

	for _, pb := range pbs {
		//Update pipeline build commits
		log.Debug("poller.ExecuterRun> get commits for pipeline build %d: %#v", pb.ID, pb)
		if commits, err := pipeline.UpdatePipelineBuildCommits(db, proj, pip, app, &sdk.DefaultEnv, &pb); err != nil {
			log.Warning("poller.ExecuterRun> Unable to update pipeline build commits")
		} else {
			log.Debug("poller.ExecuterRun> %d commits for pipeline build %d", len(commits), pb.ID)
		}
	}
}

func executerProcess(tx gorp.SqlExecutor, store cache.Store, p *sdk.RepositoryPoller, e *sdk.RepositoryPollerExecution) ([]sdk.PipelineBuild, error) {
	t := time.Now()
	e.ExecutionDate = &t
	e.Executed = true

	projectKey := p.Application.ProjectKey
	rm := p.Application.VCSServer

	proj, errProj := project.Load(tx, store, projectKey, nil)
	if errProj != nil {
		return nil, errProj
	}

	//get the client for the repositories manager
	vcsServer := repositoriesmanager.GetProjectVCSServer(proj, rm)
	client, err := repositoriesmanager.AuthorizedClient(tx, store, vcsServer)
	if err != nil {
		return nil, sdk.WrapError(err, "Polling> Unable to get client for %s %s", projectKey, rm)
	}

	//Check if the polling if disabled
	if info, err := repositoriesmanager.GetPollingInfos(client); err != nil {
		return nil, err
	} else if info.PollingDisabled || !info.PollingSupported {
		log.Info("Polling> %s polling is disabled", vcsServer.Name)
		return nil, nil
	}

	var events []interface{}
	events, pollingDelay, err = client.GetEvents(p.Application.RepositoryFullname, p.DateCreation)
	if err != nil && err.Error() != "No new events" {
		return nil, sdk.WrapError(err, "Polling> Unable to get events for %s %s", projectKey, rm)
	}
	e.PushEvents, err = client.PushEvents(p.Application.RepositoryFullname, events)
	if err != nil {
		e.Error = err.Error()
	}

	e.CreateEvents, err = client.CreateEvents(p.Application.RepositoryFullname, events)
	if err != nil {
		e.Error = err.Error()
	}

	e.DeleteEvents, err = client.DeleteEvents(p.Application.RepositoryFullname, events)
	if err != nil {
		e.Error = err.Error()
	}

	e.PullRequestEvents, err = client.PullRequestEvents(p.Application.RepositoryFullname, events)
	if err != nil {
		e.Error = err.Error()
	}

	var pbs []sdk.PipelineBuild
	if len(e.PushEvents) > 0 {
		var err error
		pbs, err = triggerPipelines(tx, store, projectKey, p, e)
		if err != nil {
			return nil, sdk.WrapError(err, "Polling> Unable to trigger pipeline %s for repository %s", p.Pipeline.Name, p.Application.RepositoryFullname)
		}
	}

	if len(e.PullRequestEvents) > 0 {
		pbsPull, errPull := triggerPipelines(tx, store, projectKey, p, e)
		if errPull != nil {
			return nil, sdk.WrapError(errPull, "Polling> Unable for pull request to trigger pipeline %s for repository %s", p.Pipeline.Name, p.Application.RepositoryFullname)
		}
		pbs = append(pbs, pbsPull...)
	}

	if err := UpdateExecution(tx, e); err != nil {
		return nil, err
	}

	return pbs, nil
}

func triggerPipelines(tx gorp.SqlExecutor, store cache.Store, projectKey string, poller *sdk.RepositoryPoller, e *sdk.RepositoryPollerExecution) ([]sdk.PipelineBuild, error) {
	proj, err := project.LoadByPipelineID(tx, store, nil, poller.Pipeline.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "Polling.triggerPipelines> Cannot load project for pipeline %s", poller.Pipeline.Name)
	}

	e.PipelineBuildVersions = map[string]int64{}

	var pbs []sdk.PipelineBuild
	for _, event := range e.PushEvents {
		pb, err := triggerPipeline(tx, poller, event, proj, false)
		if err != nil {
			return nil, sdk.WrapError(err, "Polling.triggerPipelines> cannot trigger pipeline %d", poller.Pipeline.ID)
		}

		if pb != nil {
			log.Debug("Polling.triggerPipelines> Triggered %s/%s/%s : %s", projectKey, poller.Application.RepositoryFullname, event.Branch, event.Commit.Hash)
			e.PipelineBuildVersions[event.Branch.ID+"/"+event.Commit.Hash[:7]] = pb.Version
			pbs = append(pbs, *pb)
		}
	}

	for _, event := range e.PullRequestEvents {
		pb, err := triggerPipeline(tx, poller, event.Head, proj, true)
		if err != nil {
			log.Error("Polling.triggerPipelines> cannot trigger pipeline %d: %s\n", poller.Pipeline.ID, err)
			return nil, err
		}

		if pb != nil {
			log.Debug("Polling.triggerPipelines> Triggered %s/%s/%s : %s", projectKey, poller.Application.RepositoryFullname, event.Head.Branch, event.Head.Commit.Hash)
			e.PipelineBuildVersions[event.Head.Branch.ID+"/"+event.Head.Commit.Hash[:7]] = pb.Version
			pbs = append(pbs, *pb)
		}
	}

	for _, event := range e.CreateEvents {
		pb, err := triggerPipeline(tx, poller, sdk.VCSPushEvent(event), proj, false)
		if err != nil {
			return nil, sdk.WrapError(err, "Polling.triggerPipelines> cannot trigger pipeline %d", poller.Pipeline.ID)
		}

		if pb != nil {
			log.Debug("Polling.triggerPipelines> Triggered %s/%s/%s : %s", projectKey, poller.Application.RepositoryFullname, event.Branch, event.Commit.Hash)
			e.PipelineBuildVersions[event.Branch.ID+"/"+event.Commit.Hash[:7]] = pb.Version
			pbs = append(pbs, *pb)
		}
	}

	for _, e := range e.DeleteEvents {
		if err := pipeline.DeleteBranchBuilds(tx, poller.Application.ID, e.Branch.DisplayID); err != nil {
			if err != sql.ErrNoRows {
				return nil, sdk.WrapError(err, "Polling.triggerPipelines> cannot delete pipeline build for branch %s", e.Branch.DisplayID)
			}
		}
	}

	log.Debug("Polling.triggerPipelines> %d pipelines triggered", len(pbs))

	return pbs, nil
}

func triggerPipeline(tx gorp.SqlExecutor, poller *sdk.RepositoryPoller, e sdk.VCSPushEvent, proj *sdk.Project, fork bool) (*sdk.PipelineBuild, error) {
	// Create pipeline args
	var params []sdk.Parameter

	// Load pipeline Argument
	parameters, errg := pipeline.GetAllParametersInPipeline(tx, poller.Pipeline.ID)
	if errg != nil {
		return nil, errg
	}
	poller.Pipeline.Parameter = parameters

	if fork {
		sdk.AddParameter(&params, "git.http_url", sdk.StringParameter, e.CloneURL)
		sdk.AddParameter(&params, "git.url", sdk.StringParameter, e.CloneURL)
		sdk.AddParameter(&params, "git.repository", sdk.StringParameter, e.Repo)
	}

	applicationPipelineArgs, errga := application.GetAllPipelineParam(tx, poller.Application.ID, poller.Pipeline.ID)
	if errga != nil {
		return nil, errga
	}

	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger:    false,
		VCSChangesBranch: e.Branch.ID,
		VCSChangesHash:   e.Commit.Hash,
		VCSChangesAuthor: e.Commit.Author.DisplayName,
		VCSRemoteURL:     e.CloneURL,
		VCSRemote:        e.Repo,
	}
	// Get commit message to check if we have to skip the build
	match, errm := regexp.Match(".*\\[ci skip\\].*|.*\\[cd skip\\].*", []byte(e.Commit.Message))
	if errm != nil {
		log.Warning("polling> Cannot check %s/%s for commit %s by %s : %s (%s)", proj.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor, e.Commit.Message, errm)
	}
	if match {
		log.Debug("polling> Skipping build of %s/%s for commit %s by %s", proj.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor)
		return nil, nil
	}
	//Check if build exists
	if b, err := pipeline.BuildExists(tx, poller.Application.ID, poller.Pipeline.ID, sdk.DefaultEnv.ID, &trigger); err != nil || b {
		if err != nil {
			log.Warning("Polling> Error checking existing build : %s", err)
		}
		return nil, nil
	}
	//Insert the build
	pb, err := pipeline.InsertPipelineBuild(tx, proj, &poller.Pipeline, &poller.Application, applicationPipelineArgs, params, &sdk.DefaultEnv, 0, trigger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
