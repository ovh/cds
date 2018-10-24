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
		data, err := navbar.LoadNavbarData(api.mustDB(), api.Cache, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getNavbarHandler")
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getApplicationOverviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getApplicationOverviewHandler> unable to load project")
		}

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "getApplicationOverviewHandler> unable to load application")
		}

		usage, errU := loadApplicationUsage(api.mustDB(), key, appName)
		if errU != nil {
			return sdk.WrapError(errU, "getApplicationOverviewHandler> Cannot load application usage")
		}
		app.Usage = &usage

		appOverview := sdk.ApplicationOverview{
			Graphs:  make([]sdk.ApplicationOverviewGraph, 0, 2),
			History: make(map[string][]sdk.WorkflowRun, len(app.Usage.Workflows)),
		}

		// GET METRICS
		m1, errMV := metrics.GetMetrics(api.mustDB(), key, app.ID, sdk.MetricKeyVulnerability)
		if errMV != nil {
			return sdk.WrapError(errMV, "getApplicationOverviewHandler> Cannot list vulnerability metrics")
		}
		appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
			Type:  sdk.MetricKeyVulnerability,
			Datas: m1,
		})

		m2, errUT := metrics.GetMetrics(api.mustDB(), key, app.ID, sdk.MetricKeyUnitTest)
		if errUT != nil {
			return sdk.WrapError(errUT, "getApplicationOverviewHandler> Cannot list Unit test metrics")
		}
		appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
			Type:  sdk.MetricKeyUnitTest,
			Datas: m2,
		})

		// GET VCS URL
		// Get vcs info to known if we are on the default branch or not
		if projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer); projectVCSServer != nil {
			client, erra := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projectVCSServer)
			if erra != nil {
				return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "getApplicationOverviewHandler> Cannot get repo client %s: %v", app.VCSServer, erra)
			}
			vcsRepo, errRepo := client.RepoByFullname(ctx, app.RepositoryFullname)
			if errRepo != nil {
				return sdk.WrapError(errRepo, "getApplicationOverviewHandler> unable to get repo")
			}
			appOverview.GitURL = vcsRepo.URL
			defaultBranch, errB := repositoriesmanager.DefaultBranch(ctx, client, app.RepositoryFullname)
			if errB != nil {
				return sdk.WrapError(errB, "getApplicationOverviewHandler> Unable to get default branch")
			}

			// GET LAST BUILD
			tagFilter := make(map[string]string, 1)
			tagFilter["git.branch"] = defaultBranch
			for _, w := range app.Usage.Workflows {
				runs, _, _, _, errR := workflow.LoadRuns(api.mustDB(), key, w.Name, 0, 5, tagFilter)
				if errR != nil {
					return sdk.WrapError(errR, "getApplicationOverviewHandler> Unable to load runs")
				}
				appOverview.History[w.Name] = runs
			}
		}

		return service.WriteJSON(w, appOverview, http.StatusOK)
	}
}
