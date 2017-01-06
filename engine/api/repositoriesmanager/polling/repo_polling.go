package polling

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//RunningPollers is the map of all runningPollers
var (
	RunningPollers = struct {
		Workers map[string]*Worker
		mutex   *sync.RWMutex
	}{
		Workers: map[string]*Worker{},
		mutex:   &sync.RWMutex{},
	}
	newPollerChan = make(chan *Worker)
	endPollerChan = make(chan *Worker)
)

//Worker represent a goroutine for each project responsible of repo polling
type Worker struct {
	ProjectKey string `json:"project"`
}

//NewWorker Initializes a new worker struct
func NewWorker(key string) *Worker {
	return &Worker{key}
}

//WorkerExecution represents a worker execution for a poller instance
type WorkerExecution struct {
	ID          int64              `json:"id"`
	Application string             `json:"application"`
	Pipeline    string             `json:"pipeline"`
	Execution   time.Time          `json:"execution"`
	Status      string             `json:"status"`
	Events      []sdk.VCSPushEvent `json:"events,omitempty"`
}

func isWorkerRunning(key string) bool {
	RunningPollers.mutex.RLock()
	defer RunningPollers.mutex.RUnlock()
	return RunningPollers.Workers[key] != nil
}

//Initialize all existing pollers (one poller per project)
func Initialize() {
	//This goroutine handles life of the workers
	go func() {
		for {
			select {
			case w := <-newPollerChan:
				RunningPollers.mutex.Lock()
				RunningPollers.Workers[w.ProjectKey] = w
				RunningPollers.mutex.Unlock()
				ok, quit, err := w.Poll()
				if err != nil {
					log.Warning("Polling> Unable to lauch worker %s: %s", w.ProjectKey, err)
					endPollerChan <- w
					continue
				}

				go func() {
					<-quit
					endPollerChan <- w
				}()

				if !ok {
					close(quit)
				}

			case w := <-endPollerChan:
				RunningPollers.mutex.Lock()
				delete(RunningPollers.Workers, w.ProjectKey)
				RunningPollers.mutex.Unlock()
			}
		}
	}()

	//This go routine creates (if needed) workers for all projects
	for {
		db := database.DB()
		if db == nil {
			time.Sleep(30 * time.Second)
			continue
		}

		proj, err := project.LoadAllProjects(db)
		if err != nil {
			log.Critical("Polling> Unable to load projects: %s", err)
			time.Sleep(30 * time.Second)
			continue
		}

		for _, p := range proj {
			if !isWorkerRunning(p.Key) {
				w := NewWorker(p.Key)
				newPollerChan <- w
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

//Poll initiate a poller
func (w *Worker) Poll() (bool, chan bool, error) {
	var quit chan bool
	quit = make(chan bool)

	//Check database connection
	db := database.DB()
	if db == nil {
		log.Warning("Polling> Database is unavailable")
		return false, quit, errors.New("Database is unavailable")
	}

	pollers, err := poller.LoadEnabledPollersByProject(db, w.ProjectKey)
	if err != nil {
		log.Warning("Polling> Unable to load enabled pollers")
		return false, quit, err
	}

	if len(pollers) == 0 {
		return false, quit, nil
	}

	for i := range pollers {
		p := &pollers[i]
		b, _ := repositoriesmanager.CheckApplicationIsAttached(db, p.Name, w.ProjectKey, p.Application.Name)
		if !b || p.Application.RepositoriesManager == nil || p.Application.RepositoryFullname == "" {
			continue
		}
		if !p.Application.RepositoriesManager.PollingSupported {
			log.Info("Polling is not supported by %s\n", p.Name)
			continue
		}
		log.Info("Starting poller on %s %s %s", p.Name, p.Application.Name, p.Pipeline.Name)
		go w.poll(p.Application.RepositoriesManager, p.Application.ID, p.Pipeline.ID, quit)
	}

	return true, quit, nil
}

func (w *Worker) poll(rm *sdk.RepositoriesManager, appID, pipID int64, quit chan bool) {
	delay := time.Duration(60.0 * time.Second)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var mayIWork string

	log.Debug("Polling> Start on appID=%d, pipID=%d\n", appID, pipID)

	for isWorkerRunning(w.ProjectKey) {
		//Check database connection
		db := database.DB()
		if db == nil {
			time.Sleep(60 * time.Second)
			continue
		}
		//Loading poller from database
		p, err := poller.LoadPollerByApplicationAndPipeline(db, appID, pipID)
		if err != nil {
			log.Warning("Polling> Unable to load poller appID=%d pipID=%d: %s", appID, pipID, err)
			break
		}
		//Check if poller is still enabled
		if !p.Enabled {
			log.Warning("Polling> Poller %s is disabled %s", p.Application.RepositoryFullname, err)
			break
		}

		k := cache.Key("reposmanager", "polling", w.ProjectKey, p.Application.Name, p.Pipeline.Name, p.Name)
		//If nobody is polling it
		if !cache.Get(k, &mayIWork) {
			log.Info("Polling> Polling repository %s for %s/%s  : %d\n", p.Application.RepositoryFullname, w.ProjectKey, p.Application.Name, int(delay.Seconds()))
			cache.SetWithTTL(k, "true", int(delay.Seconds()))

			e := &WorkerExecution{
				Status:    "Running",
				Execution: time.Now(),
			}

			if err := insertExecution(db, &p.Application, &p.Pipeline, e); err != nil {
				log.Warning("Polling> Unable to save execution : %s", err)
				continue
			}

			//get the client for the repositories manager
			client, err := repositoriesmanager.AuthorizedClient(db, w.ProjectKey, rm.Name)
			if err != nil {
				log.Warning("Polling> Unable to get client for %s %s : %s\n", w.ProjectKey, rm.Name, err)
				continue
			}
			var events []sdk.VCSPushEvent
			events, delay, err = client.PushEvents(p.Application.RepositoryFullname, p.DateCreation)
			if err != nil {
				log.Warning("Polling> Error with PushEvents on pipeline %s for repository %s: %s\n", p.Pipeline.Name, p.Application.RepositoryFullname, err)
				continue
			}

			if len(events) > 0 {
				s, err := triggerPipelines(db, w.ProjectKey, rm, p, events)
				if err != nil {
					log.Warning("Polling> Unable to trigger pipeline %s for repository %s\n", p.Pipeline.Name, p.Application.RepositoryFullname)
				}
				e.Status = s
			} else {
				e.Status = "No events"
			}

			e.Events = events

			if err := updateExecution(db, e); err != nil {
				log.Warning("Polling> Unable to update execution : %s", err)
			}

			//Wait for the delay
			time.Sleep(delay * time.Second)
			cache.Delete(k)
		}
		//Wait for sometime between 0 and 10 seconds
		time.Sleep(time.Duration(r.Float64()*10) * time.Second)
	}

	log.Debug("Polling> End\n")
	quit <- true
}

func triggerPipelines(db *sql.DB, projectKey string, rm *sdk.RepositoriesManager, poller *sdk.RepositoryPoller, events []sdk.VCSPushEvent) (string, error) {
	status := ""
	for _, event := range events {
		projectData, err := project.LoadProjectByPipelineID(db, poller.Pipeline.ID)
		if err != nil {
			log.Warning("Polling.triggerPipelines> Cannot load project for pipeline %s: %s\n", poller.Pipeline.Name, err)
			return "Error", err
		}

		projectsVar, err := project.GetAllVariableInProject(db, projectData.ID)
		if err != nil {
			log.Warning("Polling.triggerPipelines> Cannot load project variable: %s\n", err)
			return "Error", err
		}
		projectData.Variable = projectsVar

		//begin a tx
		tx, err := db.Begin()
		if err != nil {
			return "Error", err
		}

		ok, err := TriggerPipeline(tx, rm, poller, event, projectData)
		if err != nil {
			log.Warning("Polling.triggerPipelines> cannot trigger pipeline %d: %s\n", poller.Pipeline.ID, err)
			tx.Rollback()
			return "Error", err
		}

		// commit the tx
		if err := tx.Commit(); err != nil {
			log.Critical("Polling.triggerPipelines> Cannot commit tx; %s\n", err)
			return "Error", err
		}

		if ok {
			log.Debug("Polling.triggerPipelines> Triggered %s/%s/%s", projectKey, poller.Application.RepositoryFullname, event.Branch)
			status = fmt.Sprintf("%s Pipeline %s triggered on %s (%s)", status, poller.Pipeline.Name, event.Branch.DisplayID, event.Commit.Hash)
		} else {
			log.Info("Polling.triggerPipelines> Did not trigger %s/%s/%s\n", projectKey, poller.Application.RepositoryFullname, event.Branch.ID)
			status = fmt.Sprintf("%s Pipeline %s skipped on %s (%s)", status, poller.Pipeline.Name, event.Branch.DisplayID, event.Commit.Hash)
		}
	}

	return status, nil
}

// TriggerPipeline linked to received hook
func TriggerPipeline(tx *sql.Tx, rm *sdk.RepositoriesManager, poller *sdk.RepositoryPoller, e sdk.VCSPushEvent, projectData *sdk.Project) (bool, error) {
	client, err := repositoriesmanager.AuthorizedClient(tx, projectData.Key, rm.Name)
	if err != nil {
		return false, err
	}
	// Create pipeline args
	var args []sdk.Parameter
	args = append(args, sdk.Parameter{
		Name:  "git.branch",
		Value: e.Branch.ID,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.hash",
		Value: e.Commit.Hash,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.author",
		Value: e.Commit.Author.Name,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.repository",
		Value: poller.Application.RepositoryFullname,
	})
	args = append(args, sdk.Parameter{
		Name:  "git.project",
		Value: strings.Split(poller.Application.RepositoryFullname, "/")[0],
	})
	repo, _ := client.RepoByFullname(poller.Application.RepositoryFullname)
	if repo.SSHCloneURL != "" {
		args = append(args, sdk.Parameter{
			Name:  "git.url",
			Value: repo.SSHCloneURL,
		})
	}

	// Load pipeline Argument
	parameters, err := pipeline.GetAllParametersInPipeline(tx, poller.Pipeline.ID)
	if err != nil {
		return false, err
	}
	poller.Pipeline.Parameter = parameters

	applicationPipelineArgs, err := application.GetAllPipelineParam(tx, poller.Application.ID, poller.Pipeline.ID)
	if err != nil {
		return false, err
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
		log.Warning("polling> Cannot check %s/%s for commit %s by %s : %s (%s)\n", projectData.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor, e.Commit.Message, err)
	}
	if match {
		log.Debug("polling> Skipping build of %s/%s for commit %s by %s\n", projectData.Key, poller.Application.Name, trigger.VCSChangesHash, trigger.VCSChangesAuthor)
		return false, nil
	}

	if b, err := pipeline.BuildExists(tx, poller.Application.ID, poller.Pipeline.ID, sdk.DefaultEnv.ID, &trigger); err != nil || b {
		if err != nil {
			log.Warning("Polling> Error checking existing build : %s", err)
		}
		return false, nil
	}

	_, err = pipeline.InsertPipelineBuild(tx, projectData, &poller.Pipeline, &poller.Application, applicationPipelineArgs, args, &sdk.DefaultEnv, 0, trigger)
	if err != nil {
		return false, err
	}

	return true, nil
}

func insertExecution(db database.QueryExecuter, app *sdk.Application, pip *sdk.Pipeline, e *WorkerExecution) error {
	query := `
		insert into poller_execution (application_id, pipeline_id, execution_date, status, data)
		values($1, $2, $3, $4, $5)
		returning id
	`
	data, _ := json.Marshal(e.Events)
	if err := db.QueryRow(query, app.ID, pip.ID, e.Execution, e.Status, data).Scan(&e.ID); err != nil {
		return err
	}
	return nil
}

func updateExecution(db database.QueryExecuter, e *WorkerExecution) error {
	query := `
		update poller_execution set status = $2, data = $3 where id = $1
	`
	data, _ := json.Marshal(e.Events)
	if _, err := db.Exec(query, e.ID, e.Status, data); err != nil {
		return err
	}
	return nil
}

func deleteExecution(db database.QueryExecuter, e *WorkerExecution) error {
	query := `
		delete from poller_execution where id = $1
	`
	if _, err := db.Exec(query, e.ID); err != nil {
		return err
	}
	return nil
}

//ExecutionCleaner is  globale goroutine to remove all old polling traces
func ExecutionCleaner() {
	for {
		db := database.DB()
		if db == nil {
			time.Sleep(30 * time.Minute)
			continue
		}

		execs, _ := LoadExecutions(db, "", "")

		for i := range execs {
			fiveDaysAgo := time.Now().Add(-5 * 24 * time.Hour)
			if execs[i].Execution.Before(fiveDaysAgo) {
				deleteExecution(db, &execs[i])
			}
		}
		time.Sleep(1 * time.Hour)
	}
}

//LoadExecutions returns all executions in database
func LoadExecutions(db database.QueryExecuter, application, pipeline string) ([]WorkerExecution, error) {
	query := `
		select poller_execution.id, application.name, pipeline.name, poller_execution.execution_date, poller_execution.status, poller_execution.data
		from poller_execution, application, pipeline
		where poller_execution.application_id = application.id
		and poller_execution.pipeline_id = pipeline.id
		order by poller_execution.execution_date desc
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var es []WorkerExecution

	for rows.Next() {
		var e WorkerExecution
		var j sql.NullString

		if err := rows.Scan(&e.ID, &e.Application, &e.Pipeline, &e.Execution, &e.Status, &j); err != nil {
			return nil, err
		}
		if j.Valid {
			b := []byte(j.String)
			json.Unmarshal(b, &e.Events)
		}
		var ok = true
		if application != "" && application != e.Application {
			ok = false
		}
		if pipeline != "" && pipeline != e.Pipeline {
			ok = false
		}
		if ok {
			es = append(es, e)
		}
	}

	return es, nil
}
