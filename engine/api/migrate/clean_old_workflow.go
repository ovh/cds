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

// CleanOldWorkflow is the entry point to clean workflows
func CleanOldWorkflow(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap, apiURL string) {
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

			log.Debug("Applications to clean: %d", len(apps))
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
				go cleanApplicationHook(DBFunc(), store, &wg, *p, *a, apiURL)
				wg.Add(1)
				go cleanApplication(DBFunc(), &wg, chanErr, *a)
				wg.Add(1)
				go cleanApplicationArtifact(DBFunc(), &wg, chanErr, *a)
				wg.Add(1)
				go cleanApplicationPipelineBuild(DBFunc(), &wg, chanErr, *a)

				hasErrorChan := make(chan bool)
				go func(chanErr <-chan error, hasE chan<- bool) {
					for {
						select {
						case e, ok := <-chanErr:
							if e != nil {
								hasE <- true
								close(hasE)
								return
							}
							if !ok {
								hasE <- false
								close(hasE)
								return
							}
						}
					}
				}(chanErr, hasErrorChan)
				wg.Wait()
				close(chanErr)

				for has := range hasErrorChan {
					log.Debug("CanClean pipeline %+v", has)
					if !has {
						log.Debug("CleanOldWorkflow> Start removing pipelines")
						tx, errT := DBFunc().Begin()
						if errT != nil {
							log.Warning("CleanOldWorkflow> Cannot start transaction to clean application %s %d: %s", a.Name, a.ID, errT)
							continue
						}
						if err := application.DeleteAllApplicationPipeline(tx, a.ID); err != nil {
							log.Warning("cleanApplication>Cannot detach pipeline from application %s %d: %s", a.Name, a.ID, err)
							tx.Rollback()
							continue
						}

						a.WorkflowMigration = STATUS_DONE
						if err := application.Update(tx, store, a, u); err != nil {
							log.Warning("cleanApplication>Cannot update application migration status %s %d: %s", a.Name, a.ID, err)
							tx.Rollback()
							continue
						}

						if err := tx.Commit(); err != nil {
							log.Warning("cleanApplication>Cannot commit transaction: %s", err)
							tx.Rollback()
							continue
						}
						log.Debug("CleanOldWorkflow> End removing pipelines")
					}
					break
				}

			}

		}
	}

}

// cleanApplicationHook don't care about error
func cleanApplicationHook(db *gorp.DbMap, store cache.Store, wg *sync.WaitGroup, p sdk.Project, app sdk.Application, apiURL string) {
	log.Debug("cleanApplicationHook> Start deleting hooks")
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
	log.Debug("cleanApplicationHook> End deleting hooks")
}

func cleanApplicationArtifact(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	log.Debug("cleanApplicationArtifact> Start deleting artifacts")
	arts, err := artifact.LoadArtifactByApplicationID(db, app.ID)
	if err != nil {
		err := fmt.Errorf("cleanApplicationArtifact> Cannot load artifact for application %d: %s", app.ID, err)
		log.Warning("%s", err)
		chErr <- err
		return
	}
	for _, ar := range arts {
		if err := artifact.Delete(db, ar.ID); err != nil {
			err := fmt.Errorf("cleanApplicationArtifact> Cannot delete artifact %d : %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			continue
		}
		time.Sleep(1 * time.Second)
	}
	log.Debug("cleanApplicationArtifact> End deleting artifacts")
}

func cleanApplicationPipelineBuild(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	log.Debug("cleanApplicationPipelineBuild> Start deleting pipeline build")
	pipBuildMax := int64(50)
	for {
		// Delete test
		queryTest := `DELETE FROM pipeline_build_test WHERE pipeline_build_id IN (SELECT id FROM pipeline_build WHERE application_id = $1 ORDER BY id ASC LIMIT $2)`
		if _, err := db.Exec(queryTest, app.ID, pipBuildMax); err != nil {
			err := fmt.Errorf("cleanApplicationPipelineBuild> Cannot delete pipeline-build-test for application %d: %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			break
		}

		// Delete logs
		queryLog := `DELETE FROM pipeline_build_log WHERE pipeline_build_id IN (SELECT id FROM pipeline_build WHERE application_id = $1 ORDER BY id ASC LIMIT $2)`
		if _, err := db.Exec(queryLog, app.ID, pipBuildMax); err != nil {
			err := fmt.Errorf("cleanApplicationPipelineBuild> Cannot delete pipeline-build-log for application %d: %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			break
		}

		// Delete build job
		queryJob := `DELETE FROM pipeline_build_job WHERE pipeline_build_id IN (SELECT id FROM pipeline_build WHERE application_id = $1 ORDER BY id ASC LIMIT $2)`
		if _, err := db.Exec(queryJob, app.ID, pipBuildMax); err != nil {
			err := fmt.Errorf("cleanApplicationPipelineBuild> Cannot delete pipeline-build-job for application %d: %s", app.ID, err)
			log.Warning("%s", err)
			chErr <- err
			break
		}

		result, err := db.Exec("DELETE FROM pipeline_build where id IN (SELECT id FROM pipeline_build WHERE application_id = $1 ORDER BY id ASC LIMIT $2)", app.ID, pipBuildMax)
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
	log.Debug("cleanApplicationPipelineBuild> End deleting pipeline build")
}

func cleanApplication(db *gorp.DbMap, wg *sync.WaitGroup, chErr chan<- error, app sdk.Application) {
	defer wg.Done()
	log.Debug("cleanApplication> Start deleting scheduler/poller/trigger/warining")
	if err := scheduler.DeleteByApplicationID(db, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete scheduler for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := poller.DeleteAll(db, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete poller for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := trigger.DeleteApplicationTriggers(db, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete trigger for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}

	if err := sanity.DeleteAllApplicationWarnings(db, app.ID); err != nil {
		errF := fmt.Errorf("cleanApplication> Unable to delete warnings for application %s: %s", app.Name, err)
		log.Warning("%s", errF)
		chErr <- errF
		return
	}
	log.Debug("cleanApplication> End deleting scheduler/poller/trigger/warining")
	return
}
