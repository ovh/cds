package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		currentItemString := FormString(r, "currentItem")
		u := getUser(ctx)
		if currentItemString == "" {
			currentItemString = "0"
		}

		currentItem, errS := strconv.Atoi(currentItemString)
		if errS != nil {
			return sdk.WrapError(errS, "getTimelineHandler> Invalid format for current item")
		}

		timelineFilter, errT := user.LoadTimelineFilter(api.mustDB(), u)
		if errT != nil {
			return sdk.WrapError(errT, "getTimelineHandler> Unable to load timeline filter")
		}

		if timelineFilter.AllProjects {
			timelineFilter.Projects = make([]sdk.ProjectFilter, 0, len(u.Permissions.ProjectsPerm))
			for k := range u.Permissions.ProjectsPerm {
				timelineFilter.Projects = append(timelineFilter.Projects, sdk.ProjectFilter{Key: k})
			}
		}

		filter := sdk.EventFilter{
			CurrentItem: currentItem,
			Filter:      timelineFilter,
		}

		events, err := event.GetEvents(api.mustDB(), api.Cache, filter)
		if err != nil {
			return sdk.WrapError(err, "getTimelineHandler> Unable to load events")
		}
		return WriteJSON(w, events, http.StatusOK)
	}
}
