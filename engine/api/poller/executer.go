package poller

import (
	"regexp"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Executer is the goroutine which run the pipelines
func Executer(DBFunc func() *gorp.DbMap) {
	defer log.Error("poller.Executer> has been exited !")
	for {
		time.Sleep(5 * time.Second)
		exs, err := ExecuterRun(DBFunc())
		if err != nil {
			log.Warning("poller.Executer> Error : %s", err)
			continue
		}
		if len(exs) > 0 {
			log.Debug("poller.Executer> %d has been executed", len(exs))
		}
	}
}

//ExecuterRun is the core function of Executer goroutine
func ExecuterRun(db *gorp.DbMap) ([]sdk.RepositoryPollerExecution, error) {
	//Load pending executions
	exs, err := LoadPendingExecutions(db)
	if err != nil {
		log.Warning("poller.ExecuterRun> Unable to load pending execution : %s", err)
		return nil, err
	}

	//Process all
	for i := range exs {
		go executerRun(db, &exs[i])
		time.Sleep(10 * time.Second)
	}

	return exs, nil
}

func executerRun(db *gorp.DbMap, e *sdk.RepositoryPollerExecution) {
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
		log.Error("poller.ExecuterRun> Unable to load poller appID=%d pipID=%d: %s", e.ApplicationID, e.PipelineID, errl)
		return
	}
	pbs, err := executerProcess(tx, p, e)
	if err != nil {
		log.Error("poller.ExecuterRun> Unable to process %v : %s", e, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("poller.ExecuterRun> %s", err)
		return
	}

	//Update pipeline build commits
	app, errapp := application.LoadByID(db, e.ApplicationID, nil)
	if errapp != nil {
		log.Warning("poller.ExecuterRun> Unable to load application : %s", errapp)
		return
	}

	proj, errproj := project.Load(db, app.ProjectKey, nil)
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
		if _, err := pipeline.UpdatePipelineBuildCommits(db, proj, pip, app, &sdk.DefaultEnv, &pb); err != nil {
			log.Warning("poller.ExecuterRun> Unable to update pipeline build commits")
		}
	}
}

func executerProcess(tx gorp.SqlExecutor, p *sdk.RepositoryPoller, e *sdk.RepositoryPollerExecution) ([]sdk.PipelineBuild, error) {
	t := time.Now()
	e.ExecutionDate = &t
	e.Executed = true

	projectKey := p.Application.ProjectKey
	rm := p.Application.RepositoriesManager

	log.Debug("Polling> Get %s client for project %s", rm.Name, projectKey)

	//get the client for the repositories manager
	client, err := repositoriesmanager.AuthorizedClient(tx, projectKey, rm.Name)
	if err != nil {
		log.Warning("Polling> Unable to get client for %s %s : %s\n", projectKey, rm.Name, err)
		return nil, err
	}

	var events = []interface{}{}
	events, pollingDelay, err = client.GetEvents(p.Application.RepositoryFullname, p.DateCreation)
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
		pbs, err = triggerPipelines(tx, projectKey, rm, p, e)
		if err != nil {
			log.Warning("Polling> Unable to trigger pipeline %s for repository %s\n", p.Pipeline.Name, p.Application.RepositoryFullname)
			return nil, err
		}
	}

	if err := UpdateExecution(tx, e); err != nil {
		return nil, err
	}

	return pbs, nil
}

func triggerPipelines(tx gorp.SqlExecutor, projectKey string, rm *sdk.RepositoriesManager, poller *sdk.RepositoryPoller, e *sdk.RepositoryPollerExecution) ([]sdk.PipelineBuild, error) {
	proj, err := project.LoadByPipelineID(tx, nil, poller.Pipeline.ID)
	if err != nil {
		log.Warning("Polling.triggerPipelines> Cannot load project for pipeline %s: %s\n", poller.Pipeline.Name, err)
		return nil, err
	}

	e.PipelineBuildVersions = map[string]int64{}

	var pbs []sdk.PipelineBuild
	for _, event := range e.PushEvents {
		pb, err := triggerPipeline(tx, rm, poller, event, proj)
		if err != nil {
			log.Warning("Polling.triggerPipelines> cannot trigger pipeline %d: %s\n", poller.Pipeline.ID, err)
			return nil, err
		}

		if pb != nil {
			log.Debug("Polling.triggerPipelines> Triggered %s/%s/%s : %s", projectKey, poller.Application.RepositoryFullname, event.Branch, event.Commit.Hash)
			e.PipelineBuildVersions[event.Branch.ID+"/"+event.Commit.Hash[:7]] = pb.Version
			pbs = append(pbs, *pb)
		} else {
			log.Debug("Polling.triggerPipelines> Did not trigger %s/%s/%s\n", projectKey, poller.Application.RepositoryFullname, event.Branch.ID)
		}
	}

	return pbs, nil
}

func triggerPipeline(tx gorp.SqlExecutor, rm *sdk.RepositoriesManager, poller *sdk.RepositoryPoller, e sdk.VCSPushEvent, proj *sdk.Project) (*sdk.PipelineBuild, error) {
	// Create pipeline args
	var params []sdk.Parameter

	// Load pipeline Argument
	parameters, err := pipeline.GetAllParametersInPipeline(tx, poller.Pipeline.ID)
	if err != nil {
		return nil, err
	}
	poller.Pipeline.Parameter = parameters

	applicationPipelineArgs, err := application.GetAllPipelineParam(tx, poller.Application.ID, poller.Pipeline.ID)
	if err != nil {
		return nil, err
	}

	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger:    false,
		VCSChangesBranch: e.Branch.ID,
		VCSChangesHash:   e.Commit.Hash,
		VCSChangesAuthor: e.Commit.Author.DisplayName,
	}

	// Get commit message to check if we have to skip the build
	match, err := regexp.Match(".*\\[ci skip\\].*|.*\\[cd skip\\].*", []byte(e.Commit.Message))
	if err != nil {
		log.Warning("polling> Cannot check %s/%s for commit %s by %s : %s (%s)\n", proj.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor, e.Commit.Message, err)
	}
	if match {
		log.Debug("polling> Skipping build of %s/%s for commit %s by %s\n", proj.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor)
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
