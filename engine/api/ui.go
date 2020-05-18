package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/navbar"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getNavbarHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)
		data, err := navbar.LoadNavbarData(api.mustDB(), api.Cache, *consumer.AuthentifiedUser)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getApplicationOverviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		db := api.mustDB()

		p, errP := project.Load(db, key)
		if errP != nil {
			return sdk.WrapError(errP, "getApplicationOverviewHandler> unable to load project")
		}

		app, errA := application.LoadByName(db, key, appName)
		if errA != nil {
			return sdk.WrapError(errA, "getApplicationOverviewHandler> unable to load application")
		}

		usage, errU := loadApplicationUsage(ctx, db, key, appName)
		if errU != nil {
			return sdk.WrapError(errU, "getApplicationOverviewHandler> Cannot load application usage")
		}
		app.Usage = &usage

		appOverview := sdk.ApplicationOverview{
			Graphs:  make([]sdk.ApplicationOverviewGraph, 0, 3),
			History: make(map[string][]sdk.WorkflowRun, len(app.Usage.Workflows)),
		}

		// GET METRICS
		m1, errMV := metrics.GetMetrics(ctx, db, key, app.ID, sdk.MetricKeyVulnerability)
		if errMV != nil {
			return sdk.WrapError(errMV, "getApplicationOverviewHandler> Cannot list vulnerability metrics")
		}
		appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
			Type:  sdk.MetricKeyVulnerability,
			Datas: m1,
		})

		m2, errUT := metrics.GetMetrics(ctx, db, key, app.ID, sdk.MetricKeyUnitTest)
		if errUT != nil {
			return sdk.WrapError(errUT, "getApplicationOverviewHandler> Cannot list Unit test metrics")
		}
		appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
			Type:  sdk.MetricKeyUnitTest,
			Datas: m2,
		})

		mCov, errCov := metrics.GetMetrics(ctx, db, key, app.ID, sdk.MetricKeyCoverage)
		if errCov != nil {
			return sdk.WrapError(errCov, "cannot list coverage metrics")
		}
		appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
			Type:  sdk.MetricKeyCoverage,
			Datas: mCov,
		})

		// GET VCS URL
		// Get vcs info to known if we are on the default branch or not
		projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, api.mustDB(), p.Key, app.VCSServer)
		if err == nil {
			client, err := repositoriesmanager.AuthorizedClient(ctx, db, api.Cache, p.Key, projectVCSServer)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrNoReposManagerClientAuth,
					"cannot get repo client %s", app.VCSServer))
			}
			vcsRepo, err := client.RepoByFullname(ctx, app.RepositoryFullname)
			if err != nil {
				return sdk.WrapError(err, "unable to get repo")
			}
			appOverview.GitURL = vcsRepo.URL
			defaultBranch, err := repositoriesmanager.DefaultBranch(ctx, client, app.RepositoryFullname)
			if err != nil {
				return sdk.WrapError(err, "unable to get default branch")
			}

			// GET LAST BUILD
			tagFilter := make(map[string]string, 1)
			tagFilter["git.branch"] = defaultBranch.DisplayID
			for _, w := range app.Usage.Workflows {
				runs, _, _, _, errR := workflow.LoadRuns(db, key, w.Name, 0, 5, tagFilter)
				if errR != nil {
					return sdk.WrapError(errR, "getApplicationOverviewHandler> Unable to load runs")
				}
				appOverview.History[w.Name] = runs
			}
		}

		return service.WriteJSON(w, appOverview, http.StatusOK)
	}
}
