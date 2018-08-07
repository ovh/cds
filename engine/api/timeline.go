package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		u := getUser(ctx)
		currentItem, errS := FormInt(r, "currentItem")
		if errS != nil {
			return sdk.WrapError(errS, "getTimelineHandler> Invalid format for current item")
		}

		// Get workflow to mute
		timelineFilter, errT := user.LoadTimelineFilter(api.mustDB(), u)
		if errT != nil {
			return sdk.WrapError(errT, "getTimelineHandler> Unable to load timeline filter")
		}

		// Add all workflows to mute in a map
		muteFilter := make(map[string]bool, len(timelineFilter.Projects))
		for _, pf := range timelineFilter.Projects {
			for _, wn := range pf.WorkflowNames {
				muteFilter[pf.Key+"/"+wn] = true
			}
		}

		permToRequest := make(map[string][]string, len(u.Permissions.WorkflowsPerm))
		for k := range u.Permissions.WorkflowsPerm {
			if _, ok := muteFilter[k]; ok {
				continue
			}

			keySplitted := strings.Split(k, "/")
			pKey := keySplitted[0]
			wName := keySplitted[1]

			pFilter, ok := permToRequest[pKey]
			if !ok {
				pFilter = make([]string, 0, 1)
			}
			pFilter = append(pFilter, wName)
			permToRequest[pKey] = pFilter
		}

		request := sdk.EventFilter{
			CurrentItem: currentItem,
			Filter: sdk.TimelineFilter{
				Projects: make([]sdk.ProjectFilter, 0, len(permToRequest)),
			},
		}
		for k, v := range permToRequest {
			pFilter := sdk.ProjectFilter{
				Key:           k,
				WorkflowNames: v,
			}
			request.Filter.Projects = append(request.Filter.Projects, pFilter)
		}

		events, err := event.GetEvents(api.mustDB(), api.Cache, request)
		if err != nil {
			return sdk.WrapError(err, "getTimelineHandler> Unable to load events")
		}
		return service.WriteJSON(w, events, http.StatusOK)
	}
}
