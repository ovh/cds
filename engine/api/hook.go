package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getHookPollingVCSEvents() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		vcsServerParam := vars["vcsServer"]
		lastExec := time.Now()
		workflowID, errV := requestVarInt(r, "workflowID")
		if errV != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getHookPollingVCSEvents> cannot convert workflowID to int %s", errV)
		}

		if r.Header.Get("X-CDS-Last-Execution") != "" {
			if ts, err := strconv.ParseInt(r.Header.Get("X-CDS-Last-Execution"), 10, 64); err == nil {
				lastExec = time.Unix(0, ts)
			}
		}

		h, errL := workflow.LoadHookByUUID(api.mustDB(), uuid)
		if errL != nil {
			return sdk.WrapError(errL, "getHookPollingVCSEvents> cannot load hook")
		}
		if h == nil {
			return sdk.ErrNotFound
		}

		proj, errProj := project.Load(api.mustDB(), api.Cache, h.Config[sdk.HookConfigProject].Value, nil)
		if errProj != nil {
			return sdk.WrapError(errProj, "getHookPollingVCSEvents> cannot load project")
		}

		//get the client for the repositories manager
		vcsServer := repositoriesmanager.GetProjectVCSServer(proj, vcsServerParam)
		client, errR := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
		if errR != nil {
			return sdk.WrapError(errR, "getHookPollingVCSEvents> Unable to get client for %s %s", proj.Key, vcsServerParam)
		}

		//Check if the polling if disabled
		if info, err := repositoriesmanager.GetPollingInfos(ctx, client, *proj); err != nil {
			return sdk.WrapError(err, "cannot check if polling is enabled")
		} else if info.PollingDisabled || !info.PollingSupported {
			log.Info("getHookPollingVCSEvents> %s polling is disabled", vcsServer.Name)
			return service.WriteJSON(w, nil, http.StatusOK)
		}

		events, pollingDelay, err := client.GetEvents(ctx, h.Config["repoFullName"].Value, lastExec)
		if err != nil && err.Error() != "No new events" {
			return sdk.WrapError(err, "Unable to get events for %s %s", proj.Key, vcsServerParam)
		}
		pushEvents, err := client.PushEvents(ctx, h.Config["repoFullName"].Value, events)
		if err != nil {
			return sdk.WithStack(err)
		}

		pullRequestEvents, err := client.PullRequestEvents(ctx, h.Config["repoFullName"].Value, events)
		if err != nil {
			return sdk.WithStack(err)
		}

		repoEvents := sdk.RepositoryEvents{}
		for _, pushEvent := range pushEvents {
			exist, errB := workflow.RunExist(api.mustDB(), h.Config[sdk.HookConfigProject].Value, workflowID, pushEvent.Commit.Hash)
			if errB != nil {
				return sdk.WrapError(errB, "getHookPollingVCSEvents> Cannot check existing builds for push events")
			}
			if !exist {
				repoEvents.PushEvents = append(repoEvents.PushEvents, pushEvent)
			}
		}

		for _, pullRequestEvent := range pullRequestEvents {
			exist, errB := workflow.RunExist(api.mustDB(), h.Config[sdk.HookConfigProject].Value, workflowID, pullRequestEvent.Head.Commit.Hash)
			if errB != nil {
				return sdk.WrapError(errB, "getHookPollingVCSEvents> Cannot check existing builds for pull request events")
			}
			if !exist {
				repoEvents.PullRequestEvents = append(repoEvents.PullRequestEvents, pullRequestEvent)
			}
		}

		w.Header().Add("X-CDS-Poll-Interval", fmt.Sprintf("%.0f", pollingDelay.Seconds()))

		return service.WriteJSON(w, repoEvents, http.StatusOK)
	}
}
