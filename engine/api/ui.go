package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/navbar"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
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
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeElasticsearch)
		if err != nil {
			return err
		}

		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		app, err := application.LoadByName(ctx, tx, projectKey, appName)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}

		usage, err := loadApplicationUsage(ctx, tx, projectKey, appName)
		if err != nil {
			return sdk.WrapError(err, "cannot load application usage")
		}
		app.Usage = &usage

		appOverview := sdk.ApplicationOverview{
			Graphs:  make([]sdk.ApplicationOverviewGraph, 0, 3),
			History: make(map[string][]sdk.WorkflowRunSummary, len(app.Usage.Workflows)),
		}

		if len(srvs) > 0 {
			// Get metrics
			mVulnerability, err := metrics.GetMetrics(ctx, tx, projectKey, app.ID, sdk.MetricKeyVulnerability)
			if err != nil {
				return sdk.WrapError(err, "cannot list vulnerability metrics")
			}
			appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
				Type:  sdk.MetricKeyVulnerability,
				Datas: mVulnerability,
			})

			mTest, err := metrics.GetMetrics(ctx, tx, projectKey, app.ID, sdk.MetricKeyUnitTest)
			if err != nil {
				return sdk.WrapError(err, "cannot list Unit test metrics")
			}
			appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
				Type:  sdk.MetricKeyUnitTest,
				Datas: mTest,
			})

			mCoverage, err := metrics.GetMetrics(ctx, tx, projectKey, app.ID, sdk.MetricKeyCoverage)
			if err != nil {
				return sdk.WrapError(err, "cannot list coverage metrics")
			}
			appOverview.Graphs = append(appOverview.Graphs, sdk.ApplicationOverviewGraph{
				Type:  sdk.MetricKeyCoverage,
				Datas: mCoverage,
			})
		}

		if app.VCSServer != "" {
			// GET VCS URL
			// Get vcs info to known if we are on the default branch or not
			client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, projectKey, app.VCSServer)
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
				runs, _, _, _, err := workflow.LoadRunsSummaries(ctx, tx, projectKey, w.Name, 0, 5, tagFilter)
				if err != nil {
					return sdk.WrapError(err, "unable to load runs")
				}
				appOverview.History[w.Name] = runs
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, appOverview, http.StatusOK)
	}
}
