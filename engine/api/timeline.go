package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeElasticsearch)
		if err != nil {
			return err
		}
		if len(srvs) == 0 {
			return service.WriteJSON(w, []json.RawMessage{}, http.StatusOK)
		}

		consumer := getAPIConsumer(ctx)

		// Get index of the first element to return
		currentItem := service.FormInt(r, "currentItem")

		// Get workflow to mute
		timelineFilter, err := user.LoadTimelineFilter(api.mustDB(), consumer.AuthentifiedUser.ID)
		if err != nil {
			return sdk.WrapError(err, "unable to load timeline filter")
		}
		muteFilter := make(map[string]struct{}, len(timelineFilter.Projects))
		for _, pf := range timelineFilter.Projects {
			for _, wn := range pf.WorkflowNames {
				muteFilter[pf.Key+"/"+wn] = struct{}{}
			}
		}

		var ps []sdk.Project
		if isMaintainer(ctx) {
			ps, err = project.LoadAll(ctx, api.mustDB(), api.Cache)
		} else {
			ps, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, consumer.GetGroupIDs())
		}
		if err != nil {
			return err
		}

		ws, err := workflow.LoadAllNamesByProjectIDs(ctx, api.mustDB(), sdk.ProjectsToIDs(ps))
		if err != nil {
			return err
		}
		mWorkflowNames := map[int64][]string{}
		for i := range ws {
			if _, ok := mWorkflowNames[ws[i].ProjectID]; !ok {
				mWorkflowNames[ws[i].ProjectID] = nil
			}
			mWorkflowNames[ws[i].ProjectID] = append(mWorkflowNames[ws[i].ProjectID], ws[i].Name)
		}

		// Prepare projects filter
		filters := make([]sdk.ProjectFilter, 0, len(ps))
		for i := range ps {
			filter := sdk.ProjectFilter{
				Key: ps[i].Key,
			}
			// Add all workflow not muted to filters for current project
			if wNames, ok := mWorkflowNames[ps[i].ID]; ok {
				for j := range wNames {
					if _, ok := muteFilter[ps[i].Key+"/"+wNames[j]]; !ok {
						filter.WorkflowNames = append(filter.WorkflowNames, wNames[j])
					}
				}
			}
			if len(filter.WorkflowNames) > 0 {
				filters = append(filters, filter)
			}
		}

		request := sdk.EventFilter{
			CurrentItem: currentItem,
			Filter: sdk.TimelineFilter{
				Projects: filters,
			},
		}

		events, err := event.GetEvents(ctx, api.mustDB(), api.Cache, request)
		if err != nil {
			return sdk.WrapError(err, "unable to load events")
		}
		return service.WriteJSON(w, events, http.StatusOK)
	}
}
