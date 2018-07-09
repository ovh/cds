package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTimelineHandler() Handler {
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

		allPerm := make(map[string][]string)

		jobs := make(chan timelineFilterJob, 50)
		results := make(chan timelineFilterJob, 50)

		for w := 1; w <= 3; w++ {
			go haveToFilter(jobs, results, timelineFilter)
		}

		for k := range u.Permissions.WorkflowsPerm {
			keySplitted := strings.Split(k, "/")
			pKey := keySplitted[0]
			wName := keySplitted[1]
			j := timelineFilterJob{
				Key:          pKey,
				WorkflowName: wName,
			}
			jobs <- j
		}
		close(jobs)

		for a := 0; a < len(u.Permissions.WorkflowsPerm); a++ {
			item := <-results
			if !item.Add {
				continue
			}
			workflows, ok := allPerm[item.Key]
			if !ok {
				workflows = make([]string, 0, 1)
			}
			workflows = append(workflows, item.WorkflowName)
			allPerm[item.Key] = workflows
		}

		request := sdk.EventFilter{
			CurrentItem: currentItem,
			Filter: sdk.TimelineFilter{
				Projects: make([]sdk.ProjectFilter, 0, len(allPerm)),
			},
		}
		for k, v := range allPerm {
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
		return WriteJSON(w, events, http.StatusOK)
	}
}

type timelineFilterJob struct {
	Key          string
	WorkflowName string
	Add          bool
}

func haveToFilter(jobs <-chan timelineFilterJob, results chan<- timelineFilterJob, filter sdk.TimelineFilter) {
	for j := range jobs {
		projectFound := false
		workflowFound := false
		insert := false
	projLoop:
		for _, pf := range filter.Projects {
			if pf.Key == j.Key {
				projectFound = true
				for _, wName := range pf.WorkflowNames {
					if wName == j.WorkflowName {
						workflowFound = true
						break projLoop
					}
				}
				if !workflowFound {
					insert = true
				}
				break
			}
		}
		if !projectFound {
			insert = true
		}
		if insert {
			j.Add = true
		}

		results <- j
	}
}
