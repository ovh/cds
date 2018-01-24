package migrate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func CleanOldWorkflow(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap, apiUrl string) {
	u := &sdk.User{
		Admin:    true,
		Username: "CDS-DeleteApp",
	}
	tick := time.NewTicker(10 * time.Second).C
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting CleanOldWorkflow: %v", c.Err())
				return
			}
		case <-tick:
			apps, err := application.LoadOldApplicationWorkflowToClean(DBFunc())
			if err != nil {
				continue
			}

			for _, app := range apps {
				a, errA := application.LoadByID(DBFunc(), store, app.ID, u, application.LoadOptions.WithHooks, application.LoadOptions.WithPipelines)
				if errA != nil {
					log.Error("CleanOldWorkflow> Cannot load application %d: %s", app.ID, errA)
					continue
				}

				p, errP := project.LoadByID(DBFunc(), store, a.ProjectID, u, project.LoadOptions.WithEnvironments)
				if errP != nil {
					log.Error("CleanOldWorkflow> Cannot load project %d: %s", p.ID, errP)
					continue
				}

				chanErr := make(chan error)

				wg := sync.WaitGroup{}
				wg.Add(1)
				go cleanApplicationHook(DBFunc(), store, &wg, *p, *a, apiUrl)
				wg.Add(1)
				go cleanApplication(DBFunc(), &wg, chanErr, *a)
				wg.Add(1)
				go cleanApplicationArtifact(DBFunc(), &wg, chanErr, *a)
				wg.Add(1)
				go cleanApplicationPipelineBuild(DBFunc(), &wg, chanErr, *a)

				// TODO Read from error chan

				wg.Wait()
				close(chanErr)

				// TODO IF NO ERROR, delete app pipeline and app
				//if err := application.DeleteAllApplicationPipeline(tx, app.ID); err != nil {
				//	return sdk.WrapError(err, "cleanApplication> Unable to delete application pipeline for %s", app.Name)
				//}
				// Delete app
			}

		}
	}
	// clean application

}

// cleanApplicationHook don't care about error
func cleanApplicationHook(db *gorp.DbMap, store cache.Store, wg *sync.WaitGroup, p sdk.Project, app sdk.Application, apiURL string) {
	defer wg.Done()
	if app.VCSServer == "" {
		return
	}
	vcsServer := repositoriesmanager.GetProjectVCSServer(&p, app.VCSServer)
	if vcsServer == nil {
		return
	}
	client, err := repositoriesmanager.AuthorizedClient(db, store, vcsServer)
	if err != nil {
		log.Error("cleanApplicationHook> Cannot connect to repository manager: %s", err)
		return
	}

	t := strings.Split(app.RepositoryFullname, "/")
	if len(t) != 2 {
		log.Error("cleanApplicationHook> Application %s repository fullname is not valid %s", app.Name, app.RepositoryFullname)
		return
	}

	hooks, err := hook.LoadApplicationHooks(db, app.ID)
	if err != nil {
		log.Error("cleanApplicationHook> Cannot load hooks for application %s: %s", app.Name, err)
		return
	}

	for _, h := range hooks {
		s := apiURL + hook.HookLink
		link := fmt.Sprintf(s, h.UID, t[0], t[1])

		vcsHook := sdk.VCSHook{
			Name:     vcsServer.Name,
			URL:      link,
			Method:   "GET",
			Workflow: false,
		}

		if err := client.DeleteHook(app.RepositoryFullname, vcsHook); err != nil {
			log.Error("cleanApplicationHook> Cannot delete hooks from repomanager: %s / %s", vcsServer.Name, app.RepositoryFullname)
			return
		}

		// Delete hook on table hook is done after
	}
}

func cleanApplicationArtifact(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	arts, err := artifact.LoadArtifactByApplicationID(db, app.ID)
	if err != nil {
		err := fmt.Errorf("cleanApplicationArtifact> Cannot load artifact for application %s: %s", app.ID, err)
		log.Warning("%s", err)
		chErr <- err
		return
	}
	for _, ar := range arts {
		if err := artifact.DeleteArtifact(db, ar.ID); err != nil {
			err := fmt.Errorf("cleanApplicationArtifact> Cannot delete artifact %d : %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func cleanApplicationPipelineBuild(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	pipBuildMax := int64(50)
	for {
		result, err := db.Exec("DELETE FROM pipeline_build where application_id = $1 LIMIT $2", app.ID, pipBuildMax)
		if err != nil {
			err := fmt.Errorf("cleanApplicationPipelineBuild> Cannot delete pipeline-build for application %d: %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			break
		}
		nbRows, err := result.RowsAffected()
		if err != nil {
			err := fmt.Errorf("cleanApplicationPipelineBuild> Cannot get nb of rows affected appID %d: %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			break
		}
		if nbRows < pipBuildMax {
			break
		}
	}
}

func cleanApplication(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	tx, err := db.Begin()
	if err != nil {
		errF := fmt.Errorf("cleanApplication> Cannot start transaction to cleanApplication %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}
	defer tx.Rollback()

	if err := scheduler.DeleteByApplicationID(tx, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete scheduler for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := poller.DeleteAll(tx, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete poller for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := trigger.DeleteApplicationTriggers(tx, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete trigger for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := sanity.DeleteAllApplicationWarnings(tx, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete warnings for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}
	return
}
